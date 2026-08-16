import { createAdapterRegistry } from "./adapters/registry.js";
import type { ConfigAdapter, ParsedDocument } from "./adapters/types.js";
import type {
  ConfigValue,
  FieldSchema,
  FileSchema,
  FieldType,
  SectionSchema,
  ValidationIssue,
} from "./model.js";
import { coerceFieldValue, validateField } from "./validation.js";

export const MASKED_SECRET = "••••••••";

interface InternalFieldState {
  schema: FieldSchema;
  original: ConfigValue | undefined;
  current: ConfigValue | undefined;
  displayValue: string;
  issues: ValidationIssue[];
}

export interface FieldSnapshot {
  readonly key: string;
  readonly label: string;
  readonly type: FieldType;
  readonly displayValue: string;
  readonly sensitive: boolean;
  readonly dirty: boolean;
  readonly issues: readonly ValidationIssue[];
  readonly value?: ConfigValue;
}

export interface ChangeRecord {
  readonly key: string;
  readonly label: string;
  readonly before: ConfigValue | null | undefined;
  readonly after: ConfigValue | null | undefined;
  readonly sensitive: boolean;
  readonly restartRequired: boolean;
}

export interface EditorSnapshot {
  readonly filePath: string;
  readonly fields: Readonly<Record<string, FieldSnapshot>>;
  readonly changes: readonly ChangeRecord[];
  readonly issues: readonly ValidationIssue[];
  readonly dirty: boolean;
  readonly unstructuredChanges: boolean;
  readonly restartRequired: boolean;
}

const allFields = (schema: FileSchema): FieldSchema[] =>
  schema.sections.flatMap((section) => section.fields);

const sensitiveKey = (key: string): boolean => {
  const normalized = key.replace(/([a-z\d])([A-Z])/g, "$1.$2").toLowerCase();
  const segments = normalized.split(/[^a-z\d]+/).filter(Boolean);
  const sensitiveSegments = new Set([
    "password",
    "passwd",
    "pass",
    "secret",
    "token",
    "apikey",
    "privatekey",
    "gslt",
    "steampass",
  ]);
  return segments.some((segment) => sensitiveSegments.has(segment)) ||
    /(?:password|passwd|steampass|api[-_.]?key|private[-_.]?key|gslt)/i.test(key);
};

const inferredType = (value: ConfigValue): FieldType => {
  if (typeof value === "boolean") return "boolean";
  if (typeof value === "number") return Number.isInteger(value) ? "integer" : "number";
  if (typeof value !== "string") return "string";
  if (/^(?:true|false)$/i.test(value)) return "boolean";
  return "string";
};

const humanizeKey = (key: string): string => {
  const leaf = key.split(/[./]/).at(-1) ?? key;
  const label = leaf.replace(/[-_]+/g, " ").replace(/([a-z\d])([A-Z])/g, "$1 $2");
  return label.charAt(0).toUpperCase() + label.slice(1);
};

const schemaWithDetectedFields = (
  declaredSchema: FileSchema,
  entries: readonly { key: string; value: ConfigValue }[],
): FileSchema => {
  const declaredKeys = new Set(allFields(declaredSchema).map((field) => field.key));
  const detectedFields: FieldSchema[] = [];
  for (const entry of entries) {
    if (entry.key === "" || declaredKeys.has(entry.key)) continue;
    const sensitive = sensitiveKey(entry.key);
    detectedFields.push({
      key: entry.key,
      label: humanizeKey(entry.key),
      description: `Setting discovered in ${declaredSchema.label}.`,
      documentation:
        declaredSchema.documentation ??
        `Configuration file ${declaredSchema.path}`,
      type: sensitive ? "secret" : inferredType(entry.value),
      sensitive,
      restartRequired: true,
    });
  }
  if (detectedFields.length === 0) return declaredSchema;
  const detectedSection: SectionSchema = {
    id: "detected-settings",
    label: "Detected settings",
    description: "Settings discovered from the loaded configuration file.",
    fields: detectedFields,
  };
  return {
    ...declaredSchema,
    sections: [...declaredSchema.sections, detectedSection],
  };
};

const valuesEqual = (
  left: ConfigValue | undefined,
  right: ConfigValue | undefined,
): boolean => Object.is(left, right);

const displayValue = (
  field: FieldSchema,
  value: ConfigValue | undefined,
): string => {
  if ((field.sensitive || field.type === "secret") && value !== undefined && value !== "") {
    return MASKED_SECRET;
  }
  return value === null || value === undefined ? "" : String(value);
};

const deepFreezeSnapshot = (snapshot: EditorSnapshot): EditorSnapshot => {
  for (const field of Object.values(snapshot.fields)) {
    for (const fieldIssue of field.issues) Object.freeze(fieldIssue);
    Object.freeze(field.issues);
    Object.freeze(field);
  }
  for (const change of snapshot.changes) Object.freeze(change);
  for (const editorIssue of snapshot.issues) Object.freeze(editorIssue);
  Object.freeze(snapshot.fields);
  Object.freeze(snapshot.changes);
  Object.freeze(snapshot.issues);
  return Object.freeze(snapshot);
};

export class ConfigEditorStore {
  private readonly declaredSchema: FileSchema;
  private schemaValue: FileSchema;
  private readonly adapter: ConfigAdapter;
  private originalDocument: ParsedDocument;
  private workingDocument: ParsedDocument;
  private fields: Map<string, InternalFieldState>;

  private constructor(
    schema: FileSchema,
    document: ParsedDocument,
    adapter: ConfigAdapter,
  ) {
    this.declaredSchema = schema;
    this.adapter = adapter;
    this.schemaValue = schemaWithDetectedFields(schema, adapter.entries(document));
    this.originalDocument = document;
    this.workingDocument = document;
    this.fields = this.readFields(document, document);
  }

  get schema(): FileSchema {
    return this.schemaValue;
  }

  static fromLoadedFile(
    schema: FileSchema,
    document: ParsedDocument,
  ): ConfigEditorStore {
    if (schema.format !== document.format) {
      throw new TypeError(
        `Schema format "${schema.format}" does not match document format "${document.format}".`,
      );
    }
    const adapter = createAdapterRegistry().get(schema.format);
    if (!adapter) throw new TypeError(`No adapter registered for "${schema.format}".`);
    return new ConfigEditorStore(schema, document, adapter);
  }

  private readValue(
    document: ParsedDocument,
    field: FieldSchema,
  ): { value: ConfigValue | undefined; issues: ValidationIssue[] } {
    const raw = this.adapter.get(document, field.key);
    if (raw === undefined) return { value: undefined, issues: [] };
    try {
      const value = coerceFieldValue(field, raw);
      return { value, issues: validateField(field, value) };
    } catch (error) {
      return {
        value: undefined,
        issues: [
          {
            code: "type",
            message: error instanceof Error ? error.message : "Invalid value.",
            severity: "error",
            fieldKey: field.key,
            filePath: this.schema.path,
          },
        ],
      };
    }
  }

  private readFields(
    original: ParsedDocument,
    working: ParsedDocument,
  ): Map<string, InternalFieldState> {
    return new Map(
      allFields(this.schema).map((field) => {
        const originalResult = this.readValue(original, field);
        const workingResult = this.readValue(working, field);
        return [
          field.key,
          {
            schema: field,
            original: originalResult.value,
            current: workingResult.value,
            displayValue: displayValue(field, workingResult.value),
            issues: workingResult.issues,
          },
        ];
      }),
    );
  }

  setDisplayValue(key: string, input: string): void {
    const state = this.fields.get(key);
    if (!state) throw new TypeError(`Unknown configuration field "${key}".`);
    const sensitive = state.schema.sensitive || state.schema.type === "secret";
    if (sensitive && input === MASKED_SECRET) {
      state.displayValue = MASKED_SECRET;
      state.issues = [];
      return;
    }

    state.displayValue = input;
    let value: ConfigValue;
    try {
      value = coerceFieldValue(state.schema, input);
    } catch (error) {
      state.issues = [
        {
          code: "type",
          message: error instanceof Error ? error.message : "Invalid value.",
          severity: "error",
          fieldKey: key,
          filePath: this.schema.path,
        },
      ];
      return;
    }

    const issues = validateField(state.schema, value).map((fieldIssue) => ({
      ...fieldIssue,
      filePath: this.schema.path,
    }));
    state.issues = issues;
    if (issues.length > 0) return;
    this.workingDocument = this.adapter.set(this.workingDocument, key, value);
    state.current = value;
    if (sensitive) state.displayValue = MASKED_SECRET;
  }

  serializeSelectedFile(): string {
    return this.adapter.serialize(this.workingDocument);
  }

  serializeForDisplay(): string {
    let displayDocument = this.workingDocument;
    for (const field of allFields(this.schema)) {
      if (!(field.sensitive || field.type === "secret")) continue;
      const values = this.adapter.getAll(displayDocument, field.key);
      if (values.length === 0) continue;
      displayDocument = this.adapter.setAll(
        displayDocument,
        field.key,
        values.map(() => MASKED_SECRET),
      );
    }
    return this.adapter.serialize(displayDocument);
  }

  validateSerializedSource(source: string): ValidationIssue[] {
    const document = this.adapter.parse(source);
    const issues = this.adapter.validate(document, this.schema);
    for (const field of allFields(this.schema)) {
      issues.push(...this.readValue(document, field).issues);
    }
    return issues;
  }

  loadedSource(): string {
    return this.adapter.serialize(this.originalDocument);
  }

  replaceWorkingSource(source: string): void {
    let document = this.adapter.parse(source);
    for (const field of allFields(this.schema)) {
      if (!(field.sensitive || field.type === "secret")) continue;
      const displayedValues = this.adapter.getAll(document, field.key);
      if (!displayedValues.includes(MASKED_SECRET)) continue;
      const currentSecrets = this.adapter.getAllRaw(this.workingDocument, field.key);
      if (displayedValues.length !== currentSecrets.length) {
        throw new Error(
          `Secret occurrences for "${field.key}" cannot be added or removed in raw mode.`,
        );
      }
      document = this.adapter.setAllRaw(
        document,
        field.key,
        displayedValues.map((value, index) =>
          value === MASKED_SECRET
            ? currentSecrets[index]!
            : this.adapter.getAllRaw(document, field.key)[index]!,
        ),
      );
    }
    this.workingDocument = document;
    this.schemaValue = schemaWithDetectedFields(
      this.declaredSchema,
      this.adapter.entries(document),
    );
    this.fields = this.readFields(this.originalDocument, this.workingDocument);
  }

  acceptSavedSource(source: string): void {
    const document = this.adapter.parse(source);
    this.originalDocument = document;
    this.workingDocument = document;
    this.schemaValue = schemaWithDetectedFields(
      this.declaredSchema,
      this.adapter.entries(document),
    );
    this.fields = this.readFields(document, document);
  }

  snapshot(): EditorSnapshot {
    const fields: Record<string, FieldSnapshot> = {};
    const changes: ChangeRecord[] = [];
    const issues: ValidationIssue[] = [...this.workingDocument.issues];
    const documentDirty =
      this.adapter.serialize(this.workingDocument) !==
      this.adapter.serialize(this.originalDocument);

    for (const [key, state] of this.fields) {
      const sensitive = Boolean(state.schema.sensitive || state.schema.type === "secret");
      const dirty = !valuesEqual(state.current, state.original);
      const fieldIssues = state.issues.map((fieldIssue) => ({ ...fieldIssue }));
      const fieldSnapshot: FieldSnapshot = {
        key,
        label: state.schema.label,
        type: state.schema.type,
        displayValue: state.displayValue,
        sensitive,
        dirty,
        issues: fieldIssues,
        ...(!sensitive && state.current !== undefined
          ? { value: state.current }
          : {}),
      };
      fields[key] = fieldSnapshot;
      issues.push(...fieldIssues);

      if (dirty) {
        changes.push({
          key,
          label: state.schema.label,
          before: sensitive ? null : state.original,
          after: sensitive ? null : state.current,
          sensitive,
          restartRequired: Boolean(state.schema.restartRequired),
        });
      }
    }

    const unstructuredChanges = documentDirty && changes.length === 0;
    return deepFreezeSnapshot({
      filePath: this.schema.path,
      fields,
      changes,
      issues,
      dirty: documentDirty,
      unstructuredChanges,
      restartRequired:
        unstructuredChanges || changes.some((change) => change.restartRequired),
    });
  }
}
