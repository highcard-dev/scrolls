import { expect, test } from 'bun:test';
import { mkdtempSync, rmSync } from 'node:fs';
import { join } from 'node:path';
import { startServer } from './server';
import { PLAYER_Y } from './game';

async function connect(url: string) {
  const ws = new WebSocket(url); ws.binaryType = 'arraybuffer';
  const frames: ArrayBuffer[] = [];
  ws.addEventListener('message', e => { if (e.data instanceof ArrayBuffer) frames.push(e.data); });
  await new Promise<void>((resolve, reject) => { ws.onopen = () => resolve(); ws.onerror = () => reject(Error('connection failed')); });
  return { ws, frames };
}
async function until(check: () => boolean) {
  const end = Date.now() + 3000;
  while (!check()) { if (Date.now() > end) throw Error('Timed out waiting for game state'); await Bun.sleep(20); }
}
test('real sockets share the arena, server awards points, and SQLite survives restart', async () => {
  const dir = mkdtempSync(join(process.cwd(), '.scores-test-'));
  const database = join(dir, 'highscores.sqlite');
  let app = startServer({ port: 0, database });
  let a: Awaited<ReturnType<typeof connect>> | undefined, b: typeof a;
  try {
    const base = `http://127.0.0.1:${app.server.port}`;
    a = await connect(base.replace('http', 'ws') + '/ws?name=Alice');
    b = await connect(base.replace('http', 'ws') + '/ws?name=Bob');
    await until(() => a!.frames.some(f => new DataView(f).getUint8(8) === 2) && b!.frames.some(f => new DataView(f).getUint8(8) === 2));
    expect(new DataView(a.frames.at(-1)!).getUint8(1)).not.toBe(new DataView(b.frames.at(-1)!).getUint8(1));
    const alice = [...app.arena.players.values()].find(p => p.name === 'Alice')!;
    app.arena.enemies = [{ x: alice.x, y: PLAYER_Y - 50, row: 0 }];
    a.ws.send(new Uint8Array([4]));
    await until(() => alice.score > 0);
    expect(alice.score).toBe(40);
    a.ws.send(new Uint8Array([0]));
    expect([...app.arena.players.values()].find(p => p.name === 'Bob')!.score).toBe(0);
    const closed = new Promise<number>(resolve => b!.ws.onclose = e => resolve(e.code));
    b.ws.send('{"score":999999}'); expect(await closed).toBe(1008);
    a.ws.close(); await until(() => app.arena.players.size === 0);
    app.stop();
    app = startServer({ port: 0, database });
    const top = await (await fetch(`http://127.0.0.1:${app.server.port}/highscores`)).json();
    expect(top).toEqual([{ name: 'Alice', score: 40, wave: 1 }]);
    expect(app.arena.players.size).toBe(0);
  } finally { a?.ws.close(); b?.ws.close(); app.stop(); rmSync(dir, { recursive: true }); }
});

test('leaderboard keeps best score per display name, not the latest lower run', async () => {
  const { openScores } = await import('./scores');
  const scores = openScores(':memory:');
  try {
    scores.record({ name: 'Pilot', score: 100 }, 2);
    scores.record({ name: 'Pilot', score: 20 }, 1);
    scores.record({ name: 'Other', score: 200 }, 3);
    expect(scores.top()).toEqual([{ name: 'Other', score: 200, wave: 3 }, { name: 'Pilot', score: 100, wave: 2 }]);
  } finally { scores.close(); }
});

test('Scroll command starts the data-layer entrypoint with a persistent database path', async () => {
  const definition = Bun.YAML.parse(await Bun.file(new URL('../scroll.yaml', import.meta.url)).text()) as {
    commands: { start: { procedures: Array<{ command: string[]; env: { SCORES_DB: string } }> } };
    chunks: unknown[];
  };
  const procedure = definition.commands.start.procedures[0];
  expect(procedure.command).toEqual(['bun', 'data/server.ts']);
  expect(await Bun.file(new URL('../' + procedure.command[1], import.meta.url)).exists()).toBe(true);
  expect(procedure.env.SCORES_DB).toBe('data/highscores.sqlite');
  expect(definition.chunks).toEqual([{ name: 'server-source', path: '*.ts' }, { name: 'high-scores', path: 'highscores.sqlite*', skip_update: true }]);
});
