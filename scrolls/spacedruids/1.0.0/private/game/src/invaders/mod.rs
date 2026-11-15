mod game;
use crate::druid::openui::{animation, audio, canvas, state};
use crate::exports::druid::openui::keyboard::{EventKind, KeyEvent};
use game::{Effects, Game, Input, Phase, HEIGHT, PLAYER_Y, WIDTH};
use serde_json::json;
use std::cell::{Cell, RefCell};

const COLORS: [&str; 7] = [
    "#3c5669", "#80edcf", "#c6adff", "#ffa5bd", "#ffc781", "#f4f6e9", "#578b79",
];
const ALIEN_A: [&str; 8] = [
    "00100000100",
    "00010001000",
    "00111111100",
    "01101110110",
    "11111111111",
    "10111111101",
    "10100000101",
    "00011011000",
];
const ALIEN_B: [&str; 8] = [
    "00100000100",
    "10010001001",
    "10111111101",
    "11101110111",
    "11111111111",
    "01111111110",
    "00100000100",
    "01000000010",
];
const SHIP: [&str; 7] = [
    "0000001000000",
    "0000011100000",
    "0000011100000",
    "0111111111110",
    "1111111111111",
    "1111111111111",
    "1111111111111",
];
struct Frame {
    surface: u32,
    game: Game,
    batches: [Vec<f32>; 7],
    hud: Option<(Phase, u32, u32, u32, u32, usize)>,
}
struct Sound {
    context: u32,
    buffers: [u32; 3],
}
thread_local! {
    static FRAME: RefCell<Option<Frame>> = const { RefCell::new(None) };
    static RUNNING: Cell<bool> = const { Cell::new(false) };
    static HELD: Cell<u8> = const { Cell::new(0) };
    static SOUND: RefCell<Option<Sound>> = const { RefCell::new(None) };
    static AUDIBLE: Cell<bool> = const { Cell::new(false) };
    static AUDIO_BUSY: Cell<bool> = const { Cell::new(false) };
}
fn text(key: &str, value: &str) {
    state::set_value(key, &json!(value).to_string());
}
pub fn report(error: String) {
    text("$invadersError", &error);
}
fn rect(out: &mut Vec<f32>, x: f32, y: f32, w: f32, h: f32) {
    out.extend_from_slice(&[x, y, w, h]);
}
fn sprite(out: &mut Vec<f32>, rows: &[&str], x: f32, y: f32, size: f32) {
    let left = x - rows[0].len() as f32 * size / 2.;
    for (row, line) in rows.iter().enumerate() {
        for (col, pixel) in line.bytes().enumerate() {
            if pixel == b'1' {
                rect(
                    out,
                    left + col as f32 * size,
                    y + row as f32 * size,
                    size,
                    size,
                );
            }
        }
    }
}
fn fill(color: &str, x: f32, y: f32, w: f32, h: f32) -> canvas::DrawCommand {
    canvas::DrawCommand::FillRect(canvas::PaintedRect {
        bounds: canvas::Rect {
            x,
            y,
            width: w,
            height: h,
        },
        color: color.into(),
    })
}
fn label(value: &str, y: f32, size: u32, color: &str) -> canvas::DrawCommand {
    canvas::DrawCommand::FillText(canvas::Text {
        value: value.into(),
        at: canvas::Point {
            x: WIDTH / 2. - value.len() as f32 * size as f32 * 0.3,
            y,
        },
        font: format!("600 {size}px monospace"),
        color: color.into(),
    })
}
impl Frame {
    fn hud(&mut self) {
        let g = &self.game;
        let signature = (g.phase, g.score, g.best, g.lives, g.wave, g.remaining());
        if Some(signature) == self.hud {
            return;
        }
        self.hud = Some(signature);
        for (key, value) in [
            ("$invadersScore", format!("{:06}", g.score)),
            ("$invadersBest", format!("{:06}", g.best)),
            ("$invadersWave", format!("{:02}", g.wave)),
            ("$invadersLives", g.lives.to_string()),
            ("$invadersRemaining", g.remaining().to_string()),
        ] {
            text(key, &value);
        }
        text(
            "$invadersStatus",
            match g.phase {
                Phase::Ready => "AWAITING PILOT",
                Phase::Playing => "MISSION ACTIVE",
                Phase::Paused => "PAUSED",
                Phase::GameOver => "MISSION OVER",
            },
        );
        text(
            "$invadersPause",
            if g.phase == Phase::Paused {
                "Resume"
            } else {
                "Pause"
            },
        );
    }
    fn draw(&mut self) -> Result<(), String> {
        let g = &self.game;
        canvas::draw(
            self.surface,
            &[
                fill("#08101b", 0., 0., WIDTH, HEIGHT),
                fill("#1e3544", 25., 605., 910., 1.),
            ],
        )?;
        for batch in &mut self.batches {
            batch.clear();
        }
        for i in 0..95 {
            let x = ((i * 211 + 83) % 960) as f32;
            let y = (((i * 97) % 640) as f32 + g.time * (4. + (i % 3) as f32 * 3.)) % HEIGHT;
            rect(
                &mut self.batches[0],
                x,
                y,
                if i % 7 == 0 { 2. } else { 1. },
                2.,
            );
        }
        for e in g.enemies.iter().filter(|e| e.alive) {
            sprite(
                &mut self.batches[1 + e.row],
                if (g.time * 3.) as u32 % 2 == 0 {
                    &ALIEN_A
                } else {
                    &ALIEN_B
                },
                e.x,
                e.y - 12.,
                3.,
            );
        }
        for block in g.blocks.iter().filter(|b| b.alive) {
            rect(&mut self.batches[6], block.x, block.y, 5., 5.);
        }
        if g.phase != Phase::GameOver && (g.invulnerable <= 0. || (g.time * 12.) as u32 % 2 == 0) {
            sprite(&mut self.batches[1], &SHIP, g.x, PLAYER_Y - 9., 3.);
            if g.phase == Phase::Playing {
                rect(
                    &mut self.batches[4],
                    g.x - 9.,
                    PLAYER_Y + 13.,
                    6.,
                    5. + (g.time * 25.).sin() * 3.,
                );
                rect(
                    &mut self.batches[4],
                    g.x + 3.,
                    PLAYER_Y + 13.,
                    6.,
                    5. + (g.time * 25.).sin() * 3.,
                );
            }
        }
        for b in &g.bullets {
            rect(
                &mut self.batches[if b.enemy { 3 } else { 5 }],
                b.x - 2.,
                b.y - 7.,
                4.,
                14.,
            );
        }
        for p in &g.particles {
            rect(&mut self.batches[4], p.x, p.y, 3., 3.);
        }
        for (batch, color) in self.batches.iter().zip(COLORS) {
            if !batch.is_empty() {
                canvas::fill_rects(self.surface, batch, color)?;
            }
        }
        let overlay = match g.phase {
            Phase::Ready => Some(("READY, PILOT?", "PRESS SPACE OR START MISSION")),
            Phase::Paused => Some(("TAKE A BREATHER.", "PRESS P TO RESUME")),
            Phase::GameOver => Some(("THEY GOT THROUGH.", "PRESS SPACE TO TRY AGAIN")),
            Phase::Playing => None,
        };
        if let Some((title, hint)) = overlay {
            canvas::draw(
                self.surface,
                &[
                    fill("#08101bf2", 195., 254., 570., 145.),
                    fill("#80edcf", 440., 270., 80., 2.),
                    label(title, 324., 32, "#e9f5ec"),
                    label(hint, 366., 15, "#9cbbb6"),
                ],
            )?;
        }
        Ok(())
    }
}
pub fn init() -> Result<(), String> {
    let mut frame = Frame {
        surface: canvas::open("invaders", 960, 640)?,
        game: Game::new(),
        batches: std::array::from_fn(|_| Vec::with_capacity(4096)),
        hud: None,
    };
    frame.hud();
    frame.draw()?;
    FRAME.with(|f| f.replace(Some(frame)));
    Ok(())
}
fn input() -> Input {
    let held = HELD.get();
    Input {
        left: held & 3 != 0,
        right: held & 12 != 0,
        fire: held & 16 != 0,
    }
}
fn set_key(code: &str, pressed: bool) {
    let bit = match code {
        "KeyA" => 1,
        "ArrowLeft" => 2,
        "KeyD" => 4,
        "ArrowRight" => 8,
        "Space" => 16,
        _ => return,
    };
    HELD.set(if pressed {
        HELD.get() | bit
    } else {
        HELD.get() & !bit
    });
}
pub async fn key(event: KeyEvent) -> Result<(), String> {
    if event.kind == EventKind::Blur {
        HELD.set(0);
        return Ok(());
    }
    if event.kind == EventKind::Up {
        set_key(&event.code, false);
        return Ok(());
    }
    if event.control || event.alt || event.meta {
        return Ok(());
    }
    set_key(&event.code, true);
    if event.repeat {
        return Ok(());
    }
    match event.code.as_str() {
        "Space" => action("PlayInvaders", "{}").await,
        "KeyP" => action("PauseInvaders", "{}").await,
        "KeyR" => action("RestartInvaders", "{}").await,
        "KeyM" => action("SoundInvaders", "{}").await,
        _ => Ok(()),
    }
}
async fn start() -> Result<(), String> {
    if RUNNING.replace(true) {
        return Ok(());
    }
    let mut last = animation::next_frame().await?;
    let mut accumulator = 0.;
    loop {
        let now = animation::next_frame().await?;
        accumulator += ((now - last) / 1000.).clamp(0., 0.1) as f32;
        last = now;
        let effects = FRAME.with(|f| {
            let mut frames = f.borrow_mut();
            let frame = frames.as_mut().ok_or("Game is not initialized")?;
            let mut effects = Effects::default();
            while accumulator >= 1. / 120. {
                let fx = frame.game.step(1. / 120., input());
                accumulator -= 1. / 120.;
                effects.shot |= fx.shot;
                effects.hit |= fx.hit;
                effects.hurt |= fx.hurt;
            }
            frame.hud();
            frame.draw()?;
            Ok::<_, String>(effects)
        })?;
        if AUDIBLE.get() {
            SOUND.with(|s| {
                if let Some(sound) = s.borrow().as_ref() {
                    for (active, buffer) in [effects.shot, effects.hit, effects.hurt]
                        .into_iter()
                        .zip(sound.buffers)
                    {
                        if active {
                            audio::play(sound.context, buffer, 0., 0.12, false)?;
                        }
                    }
                }
                Ok::<_, String>(())
            })?;
        }
    }
}
impl Sound {
    fn new() -> Result<Self, String> {
        let context = audio::open(Some(48000.))?;
        let mut buffers = [0; 3];
        for (i, (duration, from, to)) in [(0.09, 900., 190.), (0.17, 230., 60.), (0.35, 130., 28.)]
            .into_iter()
            .enumerate()
        {
            let count = (48000. * duration) as u32;
            let buffer = audio::create_buffer(context, 1, count, 48000.)?;
            let samples: Vec<f32> = (0..count)
                .map(|sample| {
                    let t = sample as f32 / 48000.;
                    let progress = t / duration;
                    let phase =
                        std::f32::consts::TAU * (from * t + (to - from) * t * t / (2. * duration));
                    (if phase.sin() > 0. { 0.5 } else { -0.5 }) * (1. - progress).powi(2)
                })
                .collect();
            audio::write_channel(buffer, 0, &samples, 0)?;
            buffers[i] = buffer;
        }
        Ok(Self { context, buffers })
    }
}
async fn sound() -> Result<(), String> {
    if AUDIO_BUSY.replace(true) {
        return Ok(());
    }
    let result = async {
        let context = SOUND.with(|s| {
            let mut sound = s.borrow_mut();
            if sound.is_none() {
                *sound = Some(Sound::new()?);
            }
            Ok::<_, String>(sound.as_ref().unwrap().context)
        })?;
        if AUDIBLE.get() {
            audio::suspend(context).await?;
        } else {
            audio::resume(context).await?;
        }
        AUDIBLE.set(!AUDIBLE.get());
        text(
            "$invadersSound",
            if AUDIBLE.get() {
                "Sound on"
            } else {
                "Sound off"
            },
        );
        Ok(())
    }
    .await;
    AUDIO_BUSY.set(false);
    result
}
pub async fn action(action: &str, params: &str) -> Result<(), String> {
    let action = match action {
        "StartInvaders" => return start().await,
        "SoundInvaders" => return sound().await,
        "InputInvaders" => {
            let value: serde_json::Value =
                serde_json::from_str(params).map_err(|e| e.to_string())?;
            set_key(
                value["code"].as_str().ok_or("Missing input code")?,
                value["pressed"].as_bool().ok_or("Missing pressed state")?,
            );
            if value["pressed"] == true && value["code"] == "Space" {
                "PlayInvaders"
            } else {
                return Ok(());
            }
        }
        "MoveInvaders" => {
            let value: serde_json::Value =
                serde_json::from_str(params).map_err(|e| e.to_string())?;
            let direction = value["direction"].as_i64().ok_or("Missing direction")?;
            FRAME.with(|f| {
                if let Some(frame) = f.borrow_mut().as_mut() {
                    frame.game.step(
                        0.1,
                        Input {
                            left: direction < 0,
                            right: direction > 0,
                            fire: false,
                        },
                    );
                }
            });
            return Ok(());
        }
        "FireInvaders" => {
            FRAME.with(|f| {
                if let Some(frame) = f.borrow_mut().as_mut() {
                    frame.game.play();
                    frame.game.step(
                        1. / 120.,
                        Input {
                            left: false,
                            right: false,
                            fire: true,
                        },
                    );
                }
            });
            return start().await;
        }
        _ => action,
    };
    FRAME.with(|f| {
        if let Some(frame) = f.borrow_mut().as_mut() {
            match action {
                "PlayInvaders" => frame.game.play(),
                "RestartInvaders" => {
                    HELD.set(0);
                    frame.game.restart();
                }
                "PauseInvaders" => {
                    HELD.set(0);
                    frame.game.pause();
                }
                _ => (),
            }
            frame.hud();
        }
    });
    if action == "PlayInvaders" || action == "RestartInvaders" {
        return start().await;
    }
    Ok(())
}
