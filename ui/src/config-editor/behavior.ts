import type { ClientContext, ClientHandlers, JsonValue } from '@druid-ui/client';
import { loadEditor, saveSelectedFile, withMissingFileFallback } from './gateway.js';
import { createFileServiceGateway } from './file-service-gateway.js';
import { fingerprint } from './fingerprint.js';
import { validateManifest } from './manifest.js';
import type { ConfigEditorManifest } from './model.js';
import type { ConfigEditorStore } from './store.js';
import { copy } from './copy.js';

/** WASM owns editor behavior; the trusted host only renders JSON state and runs scoped tools. */
export function createEditorBehavior(context: ClientContext): ClientHandlers {
  let manifest: ConfigEditorManifest | undefined;
  const stores = new Map<string, ConfigEditorStore>();
  let selectedPath = '';
  let mode: 'form' | 'raw' = 'form';
  let saving = false;
  let status: string = copy.loading;
  let error: string | undefined;
  const gateway = withMissingFileFallback(createFileServiceGateway({
    async ensureFileService() { await context.callTool('ensure_file_service', { watch: false }); },
    async loadFile(path) {
      const value = await context.callTool('load_file', { path });
      if (typeof value !== 'string') throw new TypeError('File service returned non-text content.');
      return value;
    },
    async saveFile(path, content) {
      await context.callTool('save_file', { path, content });
    },
    fingerprint,
  }));
  const update = () => {
    const store = stores.get(selectedPath);
    context.setValue('$editor', JSON.parse(JSON.stringify({
      manifest, schema: store?.schema, snapshot: store?.snapshot(),
      source: store?.serializeForDisplay(), mode, saving, status, error,
    })) as JsonValue);
  };
  const select = async (path: string) => {
    if (!manifest) return;
    status = copy.loading;
    update();
    let store = stores.get(path);
    if (!store) {
      store = await loadEditor(gateway, manifest, path);
      stores.set(path, store);
    }
    selectedPath = path;
    mode = store.schema.sections.length ? 'form' : 'raw';
    status = '';
  };
  const text = (params: Record<string, JsonValue>, key: string) => {
    if (typeof params[key] !== 'string') throw new TypeError(`Missing string parameter: ${key}`);
    return params[key];
  };
  return {
    async init() {
      try {
        update();
        manifest = validateManifest(JSON.parse(await gateway.load('.druid/config-editor.manifest.json')));
        await select(manifest.files[0]!.path);
      } catch (cause) {
        error = cause instanceof Error ? cause.message : String(cause);
      }
      update();
    },
    async handleAction(action, params) {
      const store = stores.get(selectedPath);
      if (!store || saving) return;
      try {
        status = '';
        switch (action) {
          case 'SelectFile': await select(text(params, 'path')); break;
          case 'SetMode': {
            const next = text(params, 'mode');
            if (next !== 'raw' && next !== 'form') throw new TypeError('Invalid editor mode.');
            mode = next;
            break;
          }
          case 'SetField': store.setDisplayValue(text(params, 'key'), text(params, 'value')); break;
          case 'SetRaw': store.replaceWorkingSource(text(params, 'source')); break;
          case 'Save': {
            saving = true;
            status = copy.saving;
            update();
            const result = await saveSelectedFile(store, gateway);
            status = result.status === 'saved' ? copy.saved : copy.conflict;
            break;
          }
        }
      } catch (cause) {
        status = cause instanceof Error ? cause.message : String(cause);
      } finally {
        saving = false;
        update();
      }
    },
    onError(cause) {
      error = cause instanceof Error ? cause.message : String(cause);
      update();
    },
  };
}
