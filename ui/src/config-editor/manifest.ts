import type {
  ConfigEditorManifest,
  ConfigFormat,
  ConfigValue,
  FieldSchema,
  FieldType,
  FileSchema,
  SectionSchema,
  ServerSchema,
} from "./model.js";

const CONFIG_FORMATS = new Set<ConfigFormat>([
  "java-properties",
  "ini",
  "unreal-ini",
  "key-value",
  "json",
  "xml-properties",
  "raw",
]);

const FIELD_TYPES = new Set<FieldType>([
  "string",
  "integer",
  "number",
  "boolean",
  "enum",
  "secret",
]);

const DOTTED_VERSION = /^\d+(?:\.\d+)*$/;
const URL_SCHEME = /^[a-z][a-z\d+.-]*:/i;
const WINDOWS_DRIVE = /^[a-z]:\//i;

const fail = (message: string): never => {
  throw new TypeError(`Invalid config editor manifest: ${message}`);
};

const objectValue = (value: unknown, name: string): Record<string, unknown> => {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return fail(`${name} must be an object.`);
  }
  return value as Record<string, unknown>;
};

const stringValue = (value: unknown, name: string): string => {
  if (typeof value !== "string" || value.trim().length === 0) {
    return fail(`${name} must be a non-empty string.`);
  }
  return value;
};

const optionalString = (value: unknown, name: string): string | undefined => {
  if (value === undefined) return undefined;
  return stringValue(value, name);
};

const optionalBoolean = (
  value: unknown,
  name: string,
): boolean | undefined => {
  if (value === undefined) return undefined;
  if (typeof value !== "boolean") return fail(`${name} must be a boolean.`);
  return value;
};

const optionalNumber = (value: unknown, name: string): number | undefined => {
  if (value === undefined) return undefined;
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return fail(`${name} must be a finite number.`);
  }
  return value;
};

const arrayValue = (value: unknown, name: string): unknown[] => {
  if (!Array.isArray(value)) return fail(`${name} must be an array.`);
  return value;
};

const assertVersion = (value: string, name: string): void => {
  if (!DOTTED_VERSION.test(value)) {
    fail(`${name} must be a dotted numeric version.`);
  }
};

const assertSafePath = (path: string): void => {
  const segments = path.split("/");
  const unsafe =
    path.startsWith("/") ||
    path.includes("\\") ||
    path.includes("\0") ||
    URL_SCHEME.test(path) ||
    WINDOWS_DRIVE.test(path) ||
    segments.some((segment) => segment === "" || segment === "." || segment === "..");

  if (unsafe) fail(`file path "${path}" must be a safe relative path.`);
};

const validateServer = (value: unknown): ServerSchema => {
  const server = objectValue(value, "server");
  return {
    family: stringValue(server["family"], "server.family"),
    displayName: stringValue(server["displayName"], "server.displayName"),
    ...(server["appVersion"] === undefined
      ? {}
      : { appVersion: stringValue(server["appVersion"], "server.appVersion") }),
  };
};

const validateField = (value: unknown, location: string): FieldSchema => {
  const field = objectValue(value, location);
  const type = stringValue(field["type"], `${location}.type`);
  if (!FIELD_TYPES.has(type as FieldType)) {
    fail(`${location}.type "${type}" is unsupported.`);
  }

  const min = optionalNumber(field["min"], `${location}.min`);
  const max = optionalNumber(field["max"], `${location}.max`);
  if (min !== undefined && max !== undefined && min > max) {
    fail(`${location} minimum cannot exceed its maximum.`);
  }
  if ((min !== undefined || max !== undefined) && type !== "integer" && type !== "number") {
    fail(`${location} numeric ranges require an integer or number field.`);
  }

  let values: string[] | undefined;
  if (field["values"] !== undefined) {
    values = arrayValue(field["values"], `${location}.values`).map((entry, index) =>
      stringValue(entry, `${location}.values[${index}]`),
    );
    if (new Set(values).size !== values.length) {
      fail(`${location}.values contains duplicate enum values.`);
    }
  }
  if (type === "enum" && (!values || values.length === 0)) {
    fail(`${location} enum values must contain at least one value.`);
  }
  if (type !== "enum" && values !== undefined) {
    fail(`${location}.values is only valid for enum fields.`);
  }

  const pattern = optionalString(field["pattern"], `${location}.pattern`);
  if (pattern !== undefined) {
    try {
      new RegExp(pattern);
    } catch {
      fail(`${location}.pattern must be a valid regular expression.`);
    }
  }

  const since = optionalString(field["since"], `${location}.since`);
  const until = optionalString(field["until"], `${location}.until`);
  if (since !== undefined) assertVersion(since, `${location}.since`);
  if (until !== undefined) assertVersion(until, `${location}.until`);
  if (since !== undefined && until !== undefined && compareVersions(since, until) > 0) {
    fail(`${location}.since cannot be later than .until.`);
  }

  const rawDefaultValue = field["defaultValue"];
  if (
    rawDefaultValue !== undefined &&
    rawDefaultValue !== null &&
    typeof rawDefaultValue !== "string" &&
    typeof rawDefaultValue !== "number" &&
    typeof rawDefaultValue !== "boolean"
  ) {
    fail(`${location}.defaultValue must be a scalar value.`);
  }

  const result: FieldSchema = {
    key: stringValue(field["key"], `${location}.key`),
    label: stringValue(field["label"], `${location}.label`),
    description: stringValue(field["description"], `${location}.description`),
    documentation: stringValue(
      field["documentation"],
      `${location}.documentation`,
    ),
    type: type as FieldType,
  };
  if (values !== undefined) result.values = values;
  if (min !== undefined) result.min = min;
  if (max !== undefined) result.max = max;
  if (pattern !== undefined) result.pattern = pattern;
  if (rawDefaultValue !== undefined) {
    result.defaultValue = rawDefaultValue as ConfigValue;
  }
  if (field["sensitive"] !== undefined) {
    result.sensitive = optionalBoolean(field["sensitive"], `${location}.sensitive`)!;
  }
  if (field["restartRequired"] !== undefined) {
    result.restartRequired = optionalBoolean(
      field["restartRequired"],
      `${location}.restartRequired`,
    )!;
  }
  if (since !== undefined) result.since = since;
  if (until !== undefined) result.until = until;
  return result;
};

const validateSection = (value: unknown, location: string): SectionSchema => {
  const section = objectValue(value, location);
  const fields = arrayValue(section["fields"], `${location}.fields`).map((field, index) =>
    validateField(field, `${location}.fields[${index}]`),
  );
  if (fields.length === 0) fail(`${location}.fields must not be empty.`);

  const result: SectionSchema = {
    id: stringValue(section["id"], `${location}.id`),
    label: stringValue(section["label"], `${location}.label`),
    fields,
  };
  if (section["description"] !== undefined) {
    result.description = stringValue(
      section["description"],
      `${location}.description`,
    );
  }
  return result;
};

const validateFile = (value: unknown, location: string): FileSchema => {
  const file = objectValue(value, location);
  const path = stringValue(file["path"], `${location}.path`);
  assertSafePath(path);
  const format = stringValue(file["format"], `${location}.format`);
  if (!CONFIG_FORMATS.has(format as ConfigFormat)) {
    fail(`${location}.format "${format}" is unsupported.`);
  }

  const sections = arrayValue(file["sections"], `${location}.sections`).map(
    (section, index) => validateSection(section, `${location}.sections[${index}]`),
  );

  const sectionIds = new Set<string>();
  const fieldKeys = new Set<string>();
  for (const section of sections) {
    if (sectionIds.has(section.id)) {
      fail(`${location} contains duplicate section id "${section.id}".`);
    }
    sectionIds.add(section.id);
    for (const field of section.fields) {
      if (fieldKeys.has(field.key)) {
        fail(`${location} contains duplicate field key "${field.key}".`);
      }
      fieldKeys.add(field.key);
    }
  }

  const result: FileSchema = {
    path,
    format: format as ConfigFormat,
    label: stringValue(file["label"], `${location}.label`),
    sections,
  };
  if (file["description"] !== undefined) {
    result.description = stringValue(
      file["description"],
      `${location}.description`,
    );
  }
  if (file["documentation"] !== undefined) {
    result.documentation = stringValue(
      file["documentation"],
      `${location}.documentation`,
    );
  }
  return result;
};

export const validateManifest = (value: unknown): ConfigEditorManifest => {
  const manifest = objectValue(value, "manifest");
  if (manifest["version"] !== 1) fail("version must be 1.");

  const files = arrayValue(manifest["files"], "files").map((file, index) =>
    validateFile(file, `files[${index}]`),
  );
  if (files.length === 0) fail("files must not be empty.");

  const paths = new Set<string>();
  for (const file of files) {
    if (paths.has(file.path)) fail(`files contains duplicate path "${file.path}".`);
    paths.add(file.path);
  }

  return { version: 1, server: validateServer(manifest["server"]), files };
};

const compareVersions = (left: string, right: string): number => {
  const leftParts = left.split(".").map(Number);
  const rightParts = right.split(".").map(Number);
  const length = Math.max(leftParts.length, rightParts.length);
  for (let index = 0; index < length; index += 1) {
    const difference = (leftParts[index] ?? 0) - (rightParts[index] ?? 0);
    if (difference !== 0) return difference < 0 ? -1 : 1;
  }
  return 0;
};

export const fieldsForVersion = (
  file: FileSchema,
  version: string,
): FieldSchema[] => {
  assertVersion(version, "version");
  return file.sections.flatMap((section) =>
    section.fields.filter(
      (field) =>
        (field.since === undefined || compareVersions(version, field.since) >= 0) &&
        (field.until === undefined || compareVersions(version, field.until) <= 0),
    ),
  );
};
