export const WIDTH = 960, HEIGHT = 640, PLAYER_Y = 566, MAX_PLAYERS = 8;
export type Player = { id: number; name: string; x: number; score: number; lives: number; input: number; lastInput: number; cooldown: number; shield: number; respawn: number };
export type Enemy = { x: number; y: number; row: number };
export type Bullet = { x: number; y: number; owner: number };

/** One authoritative cooperative arena. Clients supply controls, never positions or scores. */
export class Arena {
  players = new Map<number, Player>();
  enemies: Enemy[] = [];
  bullets: Bullet[] = [];
  wave = 1;
  tick = 0;
  time = 0;
  direction = 1;
  enemyCooldown = 0;
  constructor(private record: (player: Player, wave: number) => void) { this.formation(); }
  formation() {
    this.enemies = Array.from({ length: 40 }, (_, i) => ({ x: 140 + i % 10 * 65, y: 90 + Math.floor(i / 10) * 43, row: Math.floor(i / 10) }));
    this.bullets = [];
    this.direction = 1;
    this.enemyCooldown = 1;
  }
  join(name: string): Player {
    const id = Array.from({ length: MAX_PLAYERS }, (_, i) => i + 1).find(id => !this.players.has(id));
    if (!id) throw Error('Arena full (8 pilots). Try again when a pilot leaves.');
    if (!this.players.size) { this.wave = 1; this.formation(); }
    const clean = name.replace(/[^\p{L}\p{N} _-]/gu, '').trim().slice(0, 24);
    const player = { id, name: clean || `Pilot ${id}`, x: 70 + (id - 1) * 115, score: 0, lives: 3, input: 0, lastInput: this.time, cooldown: 0, shield: 2, respawn: 0 };
    this.players.set(id, player);
    return player;
  }
  leave(id: number) {
    const player = this.players.get(id);
    if (player) this.record(player, this.wave);
    this.players.delete(id);
    this.bullets = this.bullets.filter(b => b.owner !== id);
  }
  input(id: number, bits: number) {
    const p = this.players.get(id);
    if (!p) return;
    p.input = bits & 15;
    p.lastInput = this.time;
    if (bits & 8 && !p.lives && this.time >= p.respawn) {
      p.lives = 3; p.score = 0; p.shield = 2;
    }
  }
  hit(p: Player) {
    if (!p.lives || p.shield > 0) return;
    p.lives--;
    p.shield = 1.5;
    if (!p.lives) { p.respawn = this.time + 3; this.record(p, this.wave); }
  }
  step(dt = 1 / 30) {
    this.time += dt; this.tick++;
    if (!this.players.size) return;
    for (const p of this.players.values()) {
      p.cooldown = Math.max(0, p.cooldown - dt);
      p.shield = Math.max(0, p.shield - dt);
      // Stop held controls if a browser disappears without a clean close.
      if (this.time - p.lastInput > 1) p.input = 0;
      if (!p.lives) continue;
      p.x = Math.max(22, Math.min(WIDTH - 22, p.x + ((p.input & 2 ? 1 : 0) - (p.input & 1 ? 1 : 0)) * 420 * dt));
      if (p.input & 4 && p.cooldown === 0) {
        this.bullets.push({ x: p.x, y: PLAYER_Y - 14, owner: p.id }); p.cooldown = 0.22;
      }
    }
    const speed = 22 + this.wave * 7 + (40 - this.enemies.length) * 0.7;
    if (this.enemies.some(e => e.x < 30 || e.x > WIDTH - 30)) {
      this.direction *= -1;
      for (const e of this.enemies) { e.y += 18; e.x = Math.max(30, Math.min(WIDTH - 30, e.x)); }
    }
    for (const e of this.enemies) e.x += this.direction * speed * dt;
    if (this.enemies.some(e => e.y > PLAYER_Y - 35)) {
      for (const p of this.players.values()) this.hit(p);
      this.wave = 1; this.formation();
    }
    this.enemyCooldown -= dt;
    if (this.enemyCooldown <= 0 && this.enemies.length) {
      const e = this.enemies[(this.tick * 17) % this.enemies.length];
      this.bullets.push({ x: e.x, y: e.y + 14, owner: 0 });
      this.enemyCooldown = Math.max(0.18, 0.7 - this.wave * 0.04);
    }
    this.bullets = this.bullets.filter(b => {
      b.y += (b.owner ? -610 : 235 + this.wave * 10) * dt;
      if (b.y < 0 || b.y > HEIGHT) return false;
      if (b.owner) {
        const index = this.enemies.findIndex(e => Math.abs(e.x - b.x) < 21 && Math.abs(e.y - b.y) < 17);
        if (index < 0) return true;
        const [enemy] = this.enemies.splice(index, 1);
        const p = this.players.get(b.owner);
        if (p) { p.score += (4 - enemy.row) * 10; this.record(p, this.wave); }
        return false;
      }
      const p = [...this.players.values()].find(p => p.lives && Math.abs(p.x - b.x) < 20 && Math.abs(PLAYER_Y - b.y) < 16);
      if (!p) return true;
      this.hit(p);
      return false;
    });
    if (!this.enemies.length) { this.wave++; this.formation(); }
  }
  /** LE protocol v1: 12-byte header, 12/player, 6/enemy, 6/projectile. */
  snapshot(you: number): Uint8Array {
    const bytes = new Uint8Array(12 + this.players.size * 12 + this.enemies.length * 6 + this.bullets.length * 6);
    const v = new DataView(bytes.buffer);
    v.setUint8(0, 1); v.setUint8(1, you); v.setUint16(2, this.wave, true); v.setUint32(4, this.tick, true);
    v.setUint8(8, this.players.size); v.setUint8(9, this.enemies.length); v.setUint16(10, this.bullets.length, true);
    let at = 12;
    for (const p of this.players.values()) {
      v.setUint8(at, p.id); v.setUint8(at + 1, p.lives); v.setUint8(at + 2, p.shield > 0 ? 1 : 0);
      v.setUint16(at + 4, Math.round(p.x), true); v.setUint16(at + 6, PLAYER_Y, true); v.setUint32(at + 8, p.score, true); at += 12;
    }
    for (const e of this.enemies) { v.setUint16(at, Math.round(e.x), true); v.setUint16(at + 2, Math.round(e.y), true); v.setUint8(at + 4, e.row); at += 6; }
    for (const b of this.bullets) { v.setUint16(at, Math.round(b.x), true); v.setUint16(at + 2, Math.round(b.y), true); v.setUint8(at + 4, b.owner); at += 6; }
    return bytes;
  }
}
