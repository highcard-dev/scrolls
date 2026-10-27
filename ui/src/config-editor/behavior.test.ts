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
    await test.behavior.handleAction('SetMode', { value: 'raw' });
    await test.behavior.handleAction('SetRaw', { value: '# raw edit\nmax-players=42\ncustom=true\n' });
    expect(test.state().source).toContain('# raw edit');
    await test.behavior.handleAction('SelectFile', { value: 'unknown.properties' });
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
  it('projects controls without exposing secrets and normalizes boolean and numeric changes', async () => {
    const test = fixture();
    const manifest = validManifest();
    test.files['.druid/config-editor.manifest.json'] = JSON.stringify({ ...manifest, files: [{
      ...manifest.files[0], sections: [{ id: 'game', label: 'Game', fields: [
        { key: 'pvp', label: 'PvP', description: 'Enable combat.', documentation: 'Server settings', type: 'boolean' },
        { key: 'difficulty', label: 'Difficulty', description: 'Game challenge.', documentation: 'Server settings', type: 'enum', values: ['easy', 'hard'] },
        { key: 'rcon.password', label: 'Password', description: 'Remote console secret.', documentation: 'Server settings', type: 'secret', sensitive: true },
        { ...manifest.files[0]!.sections[0]!.fields[0], restartRequired: true },
      ] }],
    }] });
    test.files['server.properties'] = 'pvp=false\ndifficulty=easy\nrcon.password=secret-token\nmax-players=20\n';
    await test.behavior.init?.();
    expect(test.state().error).toBeUndefined();
    expect(test.state().ready).toBe(true);
    expect(test.state().canSave).toBe(false);
    expect(JSON.stringify(test.state())).not.toContain('secret-token');
    expect(test.state().sections[0].fields).toEqual(expect.arrayContaining([
      expect.objectContaining({ key: 'pvp', type: 'boolean', checked: false }),
      expect.objectContaining({ key: 'difficulty', options: [{ value: 'easy', label: 'easy' }, { value: 'hard', label: 'hard' }] }),
      expect.objectContaining({ key: 'rcon.password', type: 'secret', value: '••••••••' }),
    ]));
    await test.behavior.handleAction('SetField', { key: 'pvp', value: true });
    await test.behavior.handleAction('SetField', { key: 'max-players', value: 30 });
    expect(test.state().canSave).toBe(true);
    expect(test.state().snapshot.restartRequired).toBe(true);
    await test.behavior.handleAction('Save', {});
    expect(test.files['server.properties']).toBe('pvp=true\ndifficulty=easy\nrcon.password=secret-token\nmax-players=30\n');
    expect(test.state().canSave).toBe(false);
  });
  it('retains unsaved edits across file switches and supports raw-only non-Minecraft files', async () => {
    const test = fixture();
    const manifest = validManifest();
    test.files['.druid/config-editor.manifest.json'] = JSON.stringify({
      ...manifest, server: { family: 'rust', displayName: 'Rust Server' },
      files: [...manifest.files, { path: 'server.cfg', label: 'Rust config', format: 'raw', sections: [] }],
    });
    test.files['server.cfg'] = 'server.hostname "My server"\n';
    await test.behavior.init?.();
    await test.behavior.handleAction('SetField', { key: 'max-players', value: '42' });
    await test.behavior.handleAction('SelectFile', { value: 'server.cfg' });
    expect(test.state().mode).toBe('raw');
    expect(test.state().tabs[0].disabled).toBe(true);
    await test.behavior.handleAction('SetRaw', { value: 'server.hostname "Edited"\n' });
    await test.behavior.handleAction('SelectFile', { value: 'server.properties' });
    expect(test.state().snapshot.fields['max-players'].displayValue).toBe('42');
    expect(test.state().canSave).toBe(true);
    await test.behavior.handleAction('SelectFile', { value: 'server.cfg' });
    expect(test.state().source).toBe('server.hostname "Edited"\n');
    await test.behavior.handleAction('Save', {});
    expect(test.files['server.cfg']).toBe('server.hostname "Edited"\n');
  });
});
