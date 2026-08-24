import { createComponent, rerender } from "@druid-ui/component";
import type { Context, Event } from "@druid-ui/component";
import { copy } from "../copy.js";
import type { FileGateway } from "../gateway.js";
import { loadEditor, saveSelectedFile } from "../gateway.js";
import { validateManifest } from "../manifest.js";
import type { ConfigEditorManifest } from "../model.js";
import type { ConfigEditorStore, EditorSnapshot } from "../store.js";
import { EDITOR_STYLES } from "../styles.js";
import { ActionBar } from "./ActionBar.js";
import { FileRail } from "./FileRail.js";
import { FormEditor } from "./FormEditor.js";
import { RawEditor } from "./RawEditor.js";

export type EditorMode = "form" | "raw";

export interface EditorAppProps {
  manifest: ConfigEditorManifest;
  store: ConfigEditorStore;
  snapshot: EditorSnapshot;
  mode: EditorMode;
  saving: boolean;
  status: string;
  onMode(mode: EditorMode): void;
  onSelect(path: string): Promise<void> | void;
  onField(key: string, value: string): void;
  onRaw(source: string): void;
  onSave(): Promise<void> | void;
}

export const EditorApp = ({
  manifest,
  store,
  snapshot,
  mode,
  saving,
  status,
  onMode,
  onSelect,
  onField,
  onRaw,
  onSave,
}: EditorAppProps) => (
  <div class="config-editor">
    <style>{EDITOR_STYLES}</style>
    <header class="editor-header">
      <div>
        <div class="eyebrow">Druid Admin UI</div>
        <h1 class="editor-title">{manifest.server.displayName}</h1>
      </div>
      <div class="header-controls">
        <div class="server-version">
          {manifest.server.appVersion ? `Version ${manifest.server.appVersion}` : copy.appTitle}
        </div>
        <ActionBar
          dirty={snapshot.dirty}
          invalid={snapshot.issues.some((issue) => issue.severity === "error")}
          restartRequired={snapshot.restartRequired}
          saving={saving}
          status={status}
          onSave={onSave}
        />
      </div>
    </header>
    <div class="editor-grid">
      <FileRail
        manifest={manifest}
        selectedPath={store.schema.path}
        onSelect={onSelect}
      />
      <main class="editor-main">
        <div class="tabs" role="tablist" aria-label="Editor mode">
          <button
            type="button"
            class="tab"
            role="tab"
            disabled={store.schema.sections.length === 0}
            aria-disabled={store.schema.sections.length === 0 ? "true" : "false"}
            aria-selected={mode === "form" ? "true" : "false"}
            onClick={() => onMode("form")}
          >
            {copy.formTab}
          </button>
          <button
            type="button"
            class="tab"
            role="tab"
            aria-selected={mode === "raw" ? "true" : "false"}
            onClick={() => onMode("raw")}
          >
            {copy.rawTab}
          </button>
        </div>
        {mode === "form" ? (
          <FormEditor schema={store.schema} snapshot={snapshot} onChange={onField} />
        ) : (
          <RawEditor source={store.serializeForDisplay()} onChange={onRaw} />
        )}
      </main>
    </div>
  </div>
);

export interface ConfigEditorComponentOptions {
  manifestPath: string;
  gateway: FileGateway;
}

export interface ConfigEditorComponent {
  init(context: Context): unknown;
  emit(nodeId: string, event: string, value: Event): void;
  asyncComplete(
    id: string,
    result: { tag: "ok" | "err"; val: unknown },
  ): void;
}

export const createConfigEditorComponent = ({
  manifestPath,
  gateway,
}: ConfigEditorComponentOptions): ConfigEditorComponent => {
  let manifest: ConfigEditorManifest | undefined;
  const stores = new Map<string, ConfigEditorStore>();
  let selectedPath = "";
  let mode: EditorMode = "form";
  let loading = false;
  let saving = false;
  let status = "";
  let errorMessage = "";

  const selectFile = async (path: string): Promise<void> => {
    if (!manifest) return;
    const previousPath = selectedPath;
    status = copy.loading;
    rerender();
    try {
      let store = stores.get(path);
      if (!store) {
        store = await loadEditor(gateway, manifest, path);
        stores.set(path, store);
      }
      selectedPath = path;
      mode = store.schema.sections.length === 0 ? "raw" : "form";
      status = "";
    } catch (error) {
      selectedPath = previousPath;
      status = error instanceof Error ? error.message : String(error);
      if (!stores.has(previousPath)) throw error;
    } finally {
      rerender();
    }
  };

  const initialise = async (): Promise<void> => {
    loading = true;
    try {
      manifest = validateManifest(JSON.parse(await gateway.load(manifestPath)));
      selectedPath = manifest.files[0]!.path;
      await selectFile(selectedPath);
    } catch (error) {
      errorMessage = error instanceof Error ? error.message : String(error);
    } finally {
      loading = false;
      rerender();
    }
  };

  const save = async (): Promise<void> => {
    const store = stores.get(selectedPath);
    if (!store) return;
    saving = true;
    status = copy.saving;
    try {
      const result = await saveSelectedFile(store, gateway);
      status = result.status === "saved" ? copy.saved : copy.conflict;
    } catch (error) {
      status = error instanceof Error ? error.message : String(error);
    } finally {
      saving = false;
      rerender();
    }
  };

  return createComponent(() => {
    if (!loading && !manifest && !errorMessage) void initialise();
    if (errorMessage) {
      return (
        <div class="config-editor">
          <style>{EDITOR_STYLES}</style>
          <div class="error-shell" role="alert">{errorMessage}</div>
        </div>
      );
    }
    const store = stores.get(selectedPath);
    if (!manifest || !store) {
      return (
        <div class="config-editor">
          <style>{EDITOR_STYLES}</style>
          <div class="status-card" aria-live="polite">{copy.loading}</div>
        </div>
      );
    }
    const snapshot = store.snapshot();
    return (
      <EditorApp
        manifest={manifest}
        store={store}
        snapshot={snapshot}
        mode={mode}
        saving={saving}
        status={status}
        onMode={(nextMode) => {
          mode = nextMode;
        }}
        onSelect={selectFile}
        onField={(key, value) => {
          store.setDisplayValue(key, value);
          status = "";
        }}
        onRaw={(source) => {
          store.replaceWorkingSource(source);
          status = "";
        }}
        onSave={save}
      />
    );
  });
};
