import { describe, expect, test } from 'bun:test';
import { Arena, PLAYER_Y } from './game';

describe('authoritative arena', () => {
  test('allocates one independent ship per connection and bounds movement', () => {
    const game = new Arena(() => {}), a = game.join('Alice'), b = game.join('Bob');
    expect(a.id).not.toBe(b.id);
    const bx = b.x;
    game.input(a.id, 2);
    for (let i = 0; i < 30; i++) game.step();
    expect(a.x).toBeGreaterThan(70);
    expect(b.x).toBe(bx);
    game.input(a.id, 3); const before = a.x; game.step(); expect(a.x).toBe(before);
    for (let i = 0; i < 120; i++) { game.input(a.id, 1); game.step(); }
    expect(a.x).toBe(22);
    expect(game.snapshot(a.id)[8]).toBe(2);
  });
  test('credits a hit only to the shooter and advances shared waves', () => {
    const recorded: number[] = [], game = new Arena(p => recorded.push(p.score));
    const a = game.join('Alice'), b = game.join('Bob');
    game.enemies = [{ x: a.x, y: PLAYER_Y - 50, row: 3 }];
    game.input(a.id, 4); game.step(); game.step();
    expect(a.score).toBe(10); expect(b.score).toBe(0);
    expect(recorded).toEqual([10]); expect(game.wave).toBe(2); expect(game.enemies.length).toBe(40);
  });
  test('death, invulnerability, respawn and disconnect remove stale controls', () => {
    const game = new Arena(() => {}), p = game.join('Alice');
    game.hit(p); expect(p.lives).toBe(3);
    for (let i = 0; i < 3; i++) { p.shield = 0; game.hit(p); }
    expect(p.lives).toBe(0); game.input(p.id, 8); expect(p.lives).toBe(0);
    game.time += 3; game.input(p.id, 8); expect(p.lives).toBe(3);
    game.input(p.id, 4); game.step(); expect(game.bullets.some(b => b.owner === p.id)).toBe(true);
    game.leave(p.id); expect(game.bullets.some(b => b.owner === p.id)).toBe(false);
    expect(game.join('New').input).toBe(0);
  });
  test('holds fire at server rate, clears stale input, and caps the arena at 8', () => {
    const game = new Arena(() => {}), p = game.join('A');
    game.input(p.id, 4);
    for (let i = 0; i < 16; i++) game.step();
    expect(game.bullets.filter(b => b.owner === p.id).length).toBeLessThanOrEqual(3);
    game.time += 2; game.step(); expect(p.input).toBe(0);
    for (let i = 0; i < 7; i++) game.join('Pilot');
    expect(() => game.join('Ninth')).toThrow('Arena full');
  });
});
