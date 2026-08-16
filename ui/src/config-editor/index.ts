export { fieldsForVersion, validateManifest } from "./manifest.js";
export { javaPropertiesAdapter } from "./adapters/java-properties.js";
export { iniAdapter, keyValueAdapter, unrealIniAdapter } from "./adapters/ini.js";
export { jsonAdapter } from "./adapters/json.js";
export { rawAdapter } from "./adapters/raw.js";
export { xmlPropertiesAdapter } from "./adapters/xml-properties.js";
export { createAdapterRegistry } from "./adapters/registry.js";
export { coerceFieldValue, validateField } from "./validation.js";
export { ConfigEditorStore, MASKED_SECRET } from "./store.js";
export { fingerprint } from "./fingerprint.js";
export {
  loadEditor,
  saveSelectedFile,
  withMissingFileFallback,
} from "./gateway.js";
export { createConfigEditorComponent, EditorApp } from "./components/EditorApp.js";
export { FileRail } from "./components/FileRail.js";
export { FieldControl } from "./components/FieldControl.js";
export { FormEditor } from "./components/FormEditor.js";
export { RawEditor } from "./components/RawEditor.js";
export { Inspector } from "./components/Inspector.js";
export { ActionBar } from "./components/ActionBar.js";
export { EDITOR_STYLES } from "./styles.js";
export { copy } from "./copy.js";
export type {
  ConfigEditorManifest,
  ConfigFormat,
  ConfigValue,
  FieldSchema,
  FieldType,
  FileSchema,
  SectionSchema,
  ServerSchema,
  ValidationIssue,
} from "./model.js";
export type {
  ConfigAdapter,
  LineNode,
  ParsedDocument,
  SourceLine,
  SourceSpan,
} from "./adapters/types.js";
export type { JavaPropertiesDocument } from "./adapters/java-properties.js";
export type { IniDocument } from "./adapters/ini.js";
export type { JsonDocument } from "./adapters/json.js";
export type { RawDocument } from "./adapters/raw.js";
export type { XmlPropertiesDocument } from "./adapters/xml-properties.js";
export type { AdapterRegistry } from "./adapters/registry.js";
export type {
  ChangeRecord,
  EditorSnapshot,
  FieldSnapshot,
} from "./store.js";
export type { FileGateway, SaveResult } from "./gateway.js";
export type {
  ConfigEditorComponentOptions,
  ConfigEditorComponent,
  EditorAppProps,
  EditorMode,
} from "./components/EditorApp.js";
