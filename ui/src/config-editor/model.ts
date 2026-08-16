export type FieldType =
  | "string"
  | "integer"
  | "number"
  | "boolean"
  | "enum"
  | "secret";

export type ConfigFormat =
  | "java-properties"
  | "ini"
  | "unreal-ini"
  | "key-value"
  | "json"
  | "xml-properties"
  | "raw";

export type ConfigValue = string | number | boolean | null;

export interface FieldSchema {
  key: string;
  label: string;
  description: string;
  documentation: string;
  type: FieldType;
  values?: string[];
  min?: number;
  max?: number;
  pattern?: string;
  defaultValue?: ConfigValue;
  sensitive?: boolean;
  restartRequired?: boolean;
  since?: string;
  until?: string;
}

export interface SectionSchema {
  id: string;
  label: string;
  description?: string;
  fields: FieldSchema[];
}

export interface FileSchema {
  path: string;
  format: ConfigFormat;
  label: string;
  description?: string;
  documentation?: string;
  sections: SectionSchema[];
}

export interface ServerSchema {
  family: string;
  displayName: string;
  appVersion?: string;
}

export interface ConfigEditorManifest {
  version: 1;
  server: ServerSchema;
  files: FileSchema[];
}

export interface ValidationIssue {
  code: string;
  message: string;
  severity: "error" | "warning";
  filePath?: string;
  fieldKey?: string;
  line?: number;
  column?: number;
}
