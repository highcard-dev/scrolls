import { Arena, MAX_PLAYERS } from './game';
import { openScores } from './scores';
import type { ServerWebSocket } from 'bun';

export function startServer({ port = 8080, database = 'highscores.sqlite' } = {}) {
  const scores = openScores(database);
  const arena = new Arena((p, wave) => scores.record(p, wave));
  const sockets = new Set<ServerWebSocket<{ id: number }>>();
  const server = Bun.serve<{ id: number }>({
    port, hostname: '0.0.0.0',
    fetch(req, server) {
      const url = new URL(req.url);
      if (url.pathname === '/health') return Response.json({ status: 'ok', players: sockets.size, wave: arena.wave });
      if (url.pathname === '/highscores') return Response.json(scores.top(), { headers: { 'Access-Control-Allow-Origin': '*' } });
      if (url.pathname !== '/ws') return new Response('SpaceDruids multiplayer backend. /ws /health /highscores', { status: 200 });
      if (arena.players.size >= MAX_PLAYERS) return new Response('Arena full (8 pilots)', { status: 503 });
      const player = arena.join(url.searchParams.get('name') ?? '');
      if (server.upgrade(req, { data: { id: player.id } })) return;
      arena.leave(player.id);
      return new Response('WebSocket upgrade required', { status: 426 });
    },
    websocket: {
      maxPayloadLength: 32,
      backpressureLimit: 64 * 1024,
      closeOnBackpressureLimit: true,
      idleTimeout: 60,
      open(ws) {
        sockets.add(ws);
        const p = arena.players.get(ws.data.id)!;
        ws.send(JSON.stringify({ type: 'welcome', id: p.id, name: p.name }));
        ws.send(JSON.stringify({ type: 'scores', scores: scores.top() }));
        ws.sendBinary(arena.snapshot(p.id));
      },
      message(ws, data) {
        if (typeof data === 'string' || data.length !== 1 || data[0] > 15) { ws.close(1008, 'Expected one control byte (0..15)'); return; }
        arena.input(ws.data.id, data[0]);
      },
      close(ws) { sockets.delete(ws); arena.leave(ws.data.id); },
    },
  });
  const tick = setInterval(() => {
    arena.step();
    for (const ws of sockets) ws.sendBinary(arena.snapshot(ws.data.id));
  }, 1000 / 30);
  let previousScores = '';
  const leaderboard = setInterval(() => {
    const next = JSON.stringify({ type: 'scores', scores: scores.top() });
    if (next !== previousScores) { previousScores = next; for (const ws of sockets) ws.send(next); }
  }, 1000);
  return { server, arena, stop() { clearInterval(tick); clearInterval(leaderboard); for (const ws of sockets) { arena.leave(ws.data.id); ws.close(1001, 'Server stopping'); } server.stop(true); scores.close(); } };
}

// Wire this deployment's public endpoint into its paired documents, then use
// Druid's existing upload operation. No Core listing is created or updated here.
async function publishConfiguredUI(root: string) {
  const endpoint = process.env.DRUID_PORT_HTTP_URL;
  if (!endpoint) return;
  const wsUrl = endpoint.replace(/^http/, 'ws') + '/ws';
  const source = await Bun.file(`${root}/private/client/src/view.openui`).text();
  const document = source.replace('$serverUrl = ""', `$serverUrl = ${JSON.stringify(wsUrl)}`);
  for (const scope of ['private', 'public']) await Bun.write(`${root}/${scope}/dist/app.openui`, document);
  const api = process.env.DRUID_API_URL;
  if (!api) return;
  for (const scope of ['private', 'public']) {
    const token = (await Bun.file(process.env.DRUID_TOKEN_FILE!).text()).trim();
    const response = await fetch(`${api}/api/v1/ui/packages/${scope}/publish`, { method: 'POST', headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }, body: '{}' });
    if (!response.ok) throw Error(`UI ${scope} upload failed (${response.status})`);
  }
  console.log('Uploaded deployment-configured private/public UI; public listing remains owner-controlled.');
}
if (import.meta.main) {
  const app = startServer({ port: Number(process.env.DRUID_PORT_HTTP ?? 8080), database: process.env.SCORES_DB ?? 'highscores.sqlite' });
  console.log(`Multiplayer arena listening on ${app.server.port}; SQLite high scores enabled.`);
  publishConfiguredUI(process.cwd()).catch(error => console.error('UI configuration upload:', error.message));
  for (const signal of ['SIGTERM', 'SIGINT'] as const) process.on(signal, () => { app.stop(); process.exit(0); });
}
