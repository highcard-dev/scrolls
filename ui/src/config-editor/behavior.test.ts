import { describe, expect, it } from 'vitest';
import type { ClientContext, JsonValue } from '@druid-ui/client';
import { createEditorBehavior } from './behavior.js';
import { validManifest } from './test-fixtures.js';

function fixture() {
  const files: Record<string, string> = {
    '.druid/config-editor.manifest.json': JSON.stringify(validManifest()),
    'server.properties': '# keep comment\nmax-players=20\ncustom=true\n',
  };
  let view: any;
  let failSave = false;
  const calls: string[] = [];
  const context: ClientContext = {
    async callTool<T extends JsonValue>(name: string, params: Record<string, JsonValue>): Promise<T> {
      calls.push(name);
      if (name === 'ensure_file_service') return 'ready' as T;
      if (name === 'load_file') {
        const value = files[String(params.path)];
        if (value === undefined) throw new Error('404 Not found');
        return value as T;
      }
      if (name === 'save_file') {
        const path = String(params.path);
        if (failSave) throw new Error('File write failed');
        files[path] = String(params.content);
        return 'saved' as T;
      }
      throw new Error(`Unexpected tool ${name}`);
    },
    getValue: () => undefined,
    setValue: (_name, value) => { view = value; },
    log: () => {},
  };
  return { behavior: createEditorBehavior(context), files, calls, state: () => view, fail: () => { failSave = true; } };
}

describe('OpenUI configuration behavior', () => {
  it('loads through the scoped host tools and edits and verifies the selected file', async () => {
    const test = fixture();
    await test.behavior.init?.();
    expect(test.calls[0]).toBe('ensure_file_service');
    expect(test.state().snapshot.dirty).toBe(false);
    await test.behavior.handleAction('SetField', { key: 'max-players', value: '30' });
    expect(test.state().snapshot.dirty).toBe(true);
    await test.behavior.handleAction('Save', {});
    expect(test.files['server.properties']).toBe('# keep comment\nmax-players=30\ncustom=true\n');
    expect(test.state().status).toBe('Saved and verified');
    expect(test.state().snapshot.dirty).toBe(false);
  });
  it('retains local edits and remote bytes when a write fails', async () => {
    const test = fixture();
    await test.behavior.init?.();
    await test.behavior.handleAction('SetField', { key: 'max-players', value: '30' });
    const original = test.files['server.properties'];
    test.fail();
    await test.behavior.handleAction('Save', {});
    expect(test.files['server.properties']).toBe(original);
    expect(test.state().snapshot.dirty).toBe(true);
    expect(test.state().status).toContain('File write failed');
    expect(test.state().saving).toBe(false);
  });
  it('retains raw editing, validation, and selected-file state on a load failure', async () => {
    const test = fixture();
    await test.behavior.init?.();
    await test.behavior.handleAction('SetMode', { mode: 'raw' });
    await test.behavior.handleAction('SetRaw', { source: '# raw edit\nmax-players=42\ncustom=true\n' });
    expect(test.state().source).toContain('# raw edit');
    await test.behavior.handleAction('SelectFile', { path: 'unknown.properties' });
    expect(test.state().schema.path).toBe('server.properties');
    await test.behavior.handleAction('SetField', { key: 'max-players', value: '-1' });
    await test.behavior.handleAction('Save', {});
    expect(test.state().status).toContain('validation errors');
    expect(test.calls.filter((call) => call === 'save_file')).toHaveLength(0);
  });
  it('surfaces startup failures instead of silently rendering an empty editor', async () => {
    const test = fixture();
    delete test.files['.druid/config-editor.manifest.json'];
    await test.behavior.init?.();
    expect(test.state().error).toContain('Missing configuration file');
  });
});
