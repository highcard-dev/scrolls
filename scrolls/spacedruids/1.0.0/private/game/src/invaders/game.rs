//! Pure simulation: no browser, WIT, clock or renderer dependencies.
pub const WIDTH: f32 = 960.;
pub const HEIGHT: f32 = 640.;
pub const PLAYER_Y: f32 = 566.;

#[derive(Clone, Copy, Debug, PartialEq)]
pub enum Phase {
    Ready,
    Playing,
    Paused,
    GameOver,
}
#[derive(Clone, Copy, Default)]
pub struct Input {
    pub left: bool,
    pub right: bool,
    pub fire: bool,
}
pub struct Enemy {
    pub x: f32,
    pub y: f32,
    pub row: usize,
    pub alive: bool,
}
pub struct Bullet {
    pub x: f32,
    pub y: f32,
    pub enemy: bool,
}
pub struct Block {
    pub x: f32,
    pub y: f32,
    pub alive: bool,
}
pub struct Particle {
    pub x: f32,
    pub y: f32,
    pub vx: f32,
    pub vy: f32,
    pub life: f32,
}
#[derive(Default)]
pub struct Effects {
    pub shot: bool,
    pub hit: bool,
    pub hurt: bool,
}
pub struct Game {
    pub phase: Phase,
    pub x: f32,
    pub score: u32,
    pub best: u32,
    pub lives: u32,
    pub wave: u32,
    pub time: f32,
    pub invulnerable: f32,
    pub enemies: Vec<Enemy>,
    pub bullets: Vec<Bullet>,
    pub blocks: Vec<Block>,
    pub particles: Vec<Particle>,
    direction: f32,
    cooldown: f32,
    enemy_cooldown: f32,
    seed: u32,
}
impl Game {
    pub fn new() -> Self {
        let mut game = Self {
            phase: Phase::Ready,
            x: WIDTH / 2.,
            score: 0,
            best: 0,
            lives: 3,
            wave: 1,
            time: 0.,
            invulnerable: 0.,
            enemies: Vec::new(),
            bullets: Vec::with_capacity(24),
            blocks: Vec::new(),
            particles: Vec::with_capacity(160),
            direction: 1.,
            cooldown: 0.,
            enemy_cooldown: 1.,
            seed: 0x51ace,
        };
        game.formation();
        game
    }
    fn formation(&mut self) {
        self.enemies = (0..40)
            .map(|i| Enemy {
                x: 140. + (i % 10) as f32 * 65.,
                y: 105. + (i / 10) as f32 * 45.,
                row: i / 10,
                alive: true,
            })
            .collect();
        self.blocks.clear();
        for base in [160., 350., 540., 730.] {
            for row in 0..7 {
                for col in 0..12 {
                    if (row == 0 && !(2..10).contains(&col)) || (row >= 4 && (4..8).contains(&col))
                    {
                        continue;
                    }
                    self.blocks.push(Block {
                        x: base + col as f32 * 6.,
                        y: 463. + row as f32 * 6.,
                        alive: true,
                    });
                }
            }
        }
        self.bullets.clear();
        self.direction = 1.;
        self.enemy_cooldown = 1.;
    }
    pub fn remaining(&self) -> usize {
        self.enemies.iter().filter(|e| e.alive).count()
    }
    pub fn play(&mut self) {
        if matches!(self.phase, Phase::Ready | Phase::GameOver) {
            self.restart();
        }
    }
    pub fn restart(&mut self) {
        let best = self.best;
        *self = Self::new();
        self.best = best;
        self.phase = Phase::Playing;
    }
    pub fn pause(&mut self) {
        self.phase = match self.phase {
            Phase::Playing => Phase::Paused,
            Phase::Paused => Phase::Playing,
            other => other,
        };
    }
    fn random(&mut self) -> u32 {
        self.seed ^= self.seed << 13;
        self.seed ^= self.seed >> 17;
        self.seed ^= self.seed << 5;
        self.seed
    }
    fn burst(&mut self, x: f32, y: f32) {
        for _ in 0..12 {
            let a = (self.random() % 628) as f32 * 0.01;
            let speed = 40. + (self.random() % 100) as f32;
            self.particles.push(Particle {
                x,
                y,
                vx: a.cos() * speed,
                vy: a.sin() * speed,
                life: 0.45,
            });
        }
    }
    pub fn step(&mut self, dt: f32, input: Input) -> Effects {
        let mut fx = Effects::default();
        if self.phase != Phase::Playing {
            return fx;
        }
        self.time += dt;
        self.invulnerable = (self.invulnerable - dt).max(0.);
        self.cooldown = (self.cooldown - dt).max(0.);
        self.x = (self.x + (input.right as i32 - input.left as i32) as f32 * 390. * dt)
            .clamp(28., WIDTH - 28.);
        if input.fire && self.cooldown <= 0. {
            self.bullets.push(Bullet {
                x: self.x,
                y: PLAYER_Y - 20.,
                enemy: false,
            });
            self.cooldown = 0.22;
            fx.shot = true;
        }
        let remaining = self.remaining();
        let speed = 22. + (40 - remaining) as f32 * 3. + (self.wave - 1) as f32 * 9.;
        let edge = self.enemies.iter().filter(|e| e.alive).any(|e| {
            e.x + self.direction * speed * dt < 26.
                || e.x + self.direction * speed * dt > WIDTH - 26.
        });
        if edge {
            self.direction = -self.direction;
        }
        for enemy in self.enemies.iter_mut().filter(|e| e.alive) {
            enemy.x += self.direction * speed * dt;
            if edge {
                enemy.y += 19.;
            }
            if enemy.y >= PLAYER_Y - 24. {
                self.phase = Phase::GameOver;
            }
        }
        self.enemy_cooldown -= dt;
        if remaining > 0
            && self.enemy_cooldown <= 0.
            && self.bullets.iter().filter(|b| b.enemy).count() < 5
        {
            let index = self.random() as usize % remaining;
            let source = self.enemies.iter().filter(|e| e.alive).nth(index).unwrap();
            self.bullets.push(Bullet {
                x: source.x,
                y: source.y + 18.,
                enemy: true,
            });
            self.enemy_cooldown = (0.9 - self.wave as f32 * 0.07).max(0.25);
        }
        let mut hits = Vec::new();
        let mut hurt = false;
        for bullet in &mut self.bullets {
            bullet.y += if bullet.enemy {
                255. + self.wave as f32 * 12.
            } else {
                -650.
            } * dt;
            if let Some(block) = self.blocks.iter().find(|b| {
                b.alive
                    && bullet.x >= b.x - 2.
                    && bullet.x <= b.x + 8.
                    && bullet.y >= b.y - 8.
                    && bullet.y <= b.y + 8.
            }) {
                let (x, y) = (block.x, block.y);
                for block in &mut self.blocks {
                    if (block.x - x).abs() <= 7. && (block.y - y).abs() <= 7. {
                        block.alive = false;
                    }
                }
                bullet.y = -100.;
                continue;
            }
            if !bullet.enemy {
                if let Some(enemy) = self.enemies.iter_mut().find(|e| {
                    e.alive && (bullet.x - e.x).abs() < 20. && (bullet.y - e.y).abs() < 16.
                }) {
                    enemy.alive = false;
                    bullet.y = -100.;
                    self.score += if enemy.row == 0 {
                        30
                    } else if enemy.row < 3 {
                        20
                    } else {
                        10
                    };
                    hits.push((enemy.x, enemy.y));
                    fx.hit = true;
                }
            } else if self.invulnerable <= 0.
                && (bullet.x - self.x).abs() < 23.
                && (bullet.y - PLAYER_Y).abs() < 17.
            {
                bullet.y = -100.;
                hurt = true;
                self.invulnerable = 1.5;
            }
        }
        self.bullets
            .retain(|b| b.y > -20. && b.y < HEIGHT + 20. && !(hurt && b.enemy));
        for (x, y) in hits {
            self.burst(x, y);
        }
        if hurt {
            self.burst(self.x, PLAYER_Y);
            self.lives -= 1;
            fx.hurt = true;
            if self.lives == 0 {
                self.phase = Phase::GameOver;
            }
        }
        self.best = self.best.max(self.score);
        for p in &mut self.particles {
            p.x += p.vx * dt;
            p.y += p.vy * dt;
            p.life -= dt;
        }
        self.particles.retain(|p| p.life > 0.);
        if self.remaining() == 0 && self.phase == Phase::Playing {
            self.wave += 1;
            self.formation();
        }
        fx
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn ready_and_paused_simulations_do_not_move() {
        let mut g = Game::new();
        let x = g.x;
        g.step(
            0.1,
            Input {
                right: true,
                fire: true,
                ..Input::default()
            },
        );
        assert_eq!(g.x, x);
        assert!(g.bullets.is_empty());
        g.play();
        g.pause();
        g.step(
            0.1,
            Input {
                right: true,
                ..Input::default()
            },
        );
        assert_eq!(g.x, x);
        assert_eq!(g.phase, Phase::Paused);
    }
    #[test]
    fn movement_is_clamped_and_opposite_keys_cancel() {
        let mut g = Game::new();
        g.play();
        for _ in 0..1000 {
            g.step(
                1. / 120.,
                Input {
                    left: true,
                    ..Input::default()
                },
            );
        }
        assert_eq!(g.x, 28.);
        g.step(
            0.1,
            Input {
                left: true,
                right: true,
                fire: false,
            },
        );
        assert_eq!(g.x, 28.);
    }
    #[test]
    fn holding_fire_obeys_cooldown() {
        let mut g = Game::new();
        g.play();
        g.blocks.clear();
        let mut shots = 0;
        for _ in 0..120 {
            shots += g
                .step(
                    1. / 120.,
                    Input {
                        fire: true,
                        ..Input::default()
                    },
                )
                .shot as usize;
        }
        assert!((4..=5).contains(&shots));
    }
    #[test]
    fn shots_award_score_once_and_remove_the_target() {
        let mut g = Game::new();
        g.play();
        g.blocks.clear();
        let e = &g.enemies[0];
        g.bullets.push(Bullet {
            x: e.x,
            y: e.y + 5.,
            enemy: false,
        });
        assert!(g.step(1. / 120., Input::default()).hit);
        assert_eq!(g.score, 30);
        assert_eq!(g.best, 30);
        assert_eq!(g.remaining(), 39);
        g.step(1. / 120., Input::default());
        assert_eq!(g.score, 30);
    }
    #[test]
    fn shields_are_destructible_and_block_projectiles() {
        let mut g = Game::new();
        g.play();
        let b = &g.blocks[0];
        g.bullets.push(Bullet {
            x: b.x + 3.,
            y: b.y,
            enemy: true,
        });
        g.step(1. / 120., Input::default());
        assert!(g.blocks.iter().any(|b| !b.alive));
        assert!(g.bullets.is_empty());
    }
    #[test]
    fn player_hit_grants_invulnerability_and_three_hits_end_game() {
        let mut g = Game::new();
        g.play();
        g.blocks.clear();
        for lives in (0..3).rev() {
            g.invulnerable = 0.;
            for _ in 0..2 {
                g.bullets.push(Bullet {
                    x: g.x,
                    y: PLAYER_Y,
                    enemy: true,
                });
            }
            g.step(1. / 120., Input::default());
            assert_eq!(g.lives, lives);
        }
        assert_eq!(g.phase, Phase::GameOver);
        g.play();
        assert_eq!(g.lives, 3);
        assert_eq!(g.phase, Phase::Playing);
    }
    #[test]
    fn clearing_wave_keeps_score_and_advances_formation() {
        let mut g = Game::new();
        g.play();
        g.score = 800;
        for e in &mut g.enemies {
            e.alive = false;
        }
        g.step(1. / 120., Input::default());
        assert_eq!(g.wave, 2);
        assert_eq!(g.remaining(), 40);
        assert_eq!(g.score, 800);
    }
    #[test]
    fn landing_is_game_over_and_restart_preserves_best_only() {
        let mut g = Game::new();
        g.play();
        g.score = 200;
        g.best = 200;
        g.enemies[0].y = PLAYER_Y;
        g.step(1. / 120., Input::default());
        assert_eq!(g.phase, Phase::GameOver);
        g.restart();
        assert_eq!(g.score, 0);
        assert_eq!(g.best, 200);
        assert_eq!(g.wave, 1);
    }
}
