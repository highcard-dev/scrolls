import type {
  ConfigFormat,
  ConfigValue,
  FileSchema,
  ValidationIssue,
} from "../model.js";

export interface SourceSpan {
  start: number;
  end: number;
}

export interface SourceLine {
  raw: string;
  ending: "\n" | "\r\n" | "";
  start: number;
  end: number;
}

export interface LineNode extends SourceLine {
  kind:
    | "blank"
    | "comment"
    | "section"
    | "entry"
    | "continuation"
    | "invalid";
  key?: string;
  value?: string;
  valueSpan?: SourceSpan;
}

export interface ParsedDocument {
  format: ConfigFormat;
  source: string;
  nodes: readonly LineNode[];
  issues: readonly ValidationIssue[];
}

export interface ConfigEntry {
  key: string;
  value: ConfigValue;
}

export interface ConfigAdapter<TDocument extends ParsedDocument = ParsedDocument> {
  parse(source: string): TDocument;
  entries(document: TDocument): readonly ConfigEntry[];
  get(document: TDocument, key: string): ConfigValue | undefined;
  getAll(document: TDocument, key: string): readonly ConfigValue[];
  getAllRaw(document: TDocument, key: string): readonly string[];
  set(document: TDocument, key: string, value: ConfigValue): TDocument;
  setAll(
    document: TDocument,
    key: string,
    values: readonly ConfigValue[],
  ): TDocument;
  setAllRaw(document: TDocument, key: string, values: readonly string[]): TDocument;
  validate(document: TDocument, schema: FileSchema): ValidationIssue[];
  serialize(document: TDocument): string;
}
