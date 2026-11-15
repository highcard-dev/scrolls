import { Database } from 'bun:sqlite';
import { mkdirSync } from 'node:fs';
import { dirname } from 'node:path';
import type { Player } from './game';

export function openScores(path: string) {
  mkdirSync(dirname(path), { recursive: true });
  const db = new Database(path, { create: true });
  db.exec(`PRAGMA journal_mode = WAL;
    CREATE TABLE IF NOT EXISTS high_scores (name TEXT PRIMARY KEY, score INTEGER NOT NULL, wave INTEGER NOT NULL);`);
  const put = db.prepare(`INSERT INTO high_scores (name, score, wave) VALUES (?, ?, ?)
    ON CONFLICT(name) DO UPDATE SET score=excluded.score, wave=excluded.wave WHERE excluded.score > high_scores.score`);
  const top = db.prepare('SELECT name, score, wave FROM high_scores ORDER BY score DESC, name ASC LIMIT 10');
  return {
    record(player: Pick<Player, 'name' | 'score'>, wave: number) { if (player.score > 0) put.run(player.name, player.score, wave); },
    top: () => top.all(),
    close: () => db.close(),
  };
}
