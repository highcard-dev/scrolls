use crate::{
    druid::{
        browser::websocket,
        openui::{animation, canvas, state},
    },
    protocol::Snapshot,
};
use serde_json::{json, Value};
use std::{
    cell::{Cell, RefCell},
    future::{poll_fn, Future},
    pin::pin,
};

const COLORS: [&str; 8] = [
    "#80edcf", "#c6adff", "#ffa5bd", "#ffc781", "#73bfff", "#f4f6e9", "#ff9566", "#d8ed80",
];
const ALIEN: [&str; 8] = [
    "00100000100",
    "00010001000",
    "00111111100",
    "01101110110",
    "11111111111",
    "10111111101",
    "10100000101",
    "00011011000",
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
struct View {
    surface: u32,
    current: Snapshot,
    previous: Snapshot,
    received: f64,
    batches: [Vec<f32>; 8],
    hud: String,
}
thread_local! {
    static VIEW: RefCell<Option<View>> = const { RefCell::new(None) };
    static SOCKET: Cell<Option<u32>> = const { Cell::new(None) };
    static HELD: Cell<u8> = const { Cell::new(0) };
    static BUSY: Cell<bool> = const { Cell::new(false) };
    static TAP: Cell<u8> = const { Cell::new(0) };
}
pub struct Client;
fn set(key: &str, value: impl Into<Value>) {
    state::set_value(key, &value.into().to_string());
}
fn get(key: &str) -> String {
    state::get_value(key)
        .and_then(|v| serde_json::from_str::<String>(&v).ok())
        .unwrap_or_default()
}
fn input() -> u8 {
    let k = HELD.get();
    (if k & 3 != 0 { 1 } else { 0 })
        | (if k & 12 != 0 { 2 } else { 0 })
        | (if k & 16 != 0 { 4 } else { 0 })
}
fn send(bits: u8) -> Result<(), String> {
    if let Some(socket) = SOCKET.get() {
        websocket::send(socket, &websocket::Frame::Binary(vec![bits]))?;
    }
    Ok(())
}
fn rect(out: &mut Vec<f32>, x: f32, y: f32, w: f32, h: f32) {
    out.extend_from_slice(&[x, y, w, h]);
}
fn sprite(out: &mut Vec<f32>, rows: &[&str], x: f32, y: f32, scale: f32) {
    for (row, line) in rows.iter().enumerate() {
        for (col, pixel) in line.bytes().enumerate() {
            if pixel == b'1' {
                rect(
                    out,
                    x + (col as f32 - rows[0].len() as f32 / 2.) * scale,
                    y + row as f32 * scale,
                    scale,
                    scale,
                );
            }
        }
    }
}
impl View {
    fn draw(&mut self, now: f64) -> Result<(), String> {
        canvas::draw(
            self.surface,
            &[canvas::DrawCommand::FillRect(canvas::PaintedRect {
                bounds: canvas::Rect {
                    x: 0.,
                    y: 0.,
                    width: 960.,
                    height: 640.,
                },
                color: "#08111d".into(),
            })],
        )?;
        for batch in &mut self.batches {
            batch.clear();
        }
        for i in 0..70 {
            rect(
                &mut self.batches[5],
                ((i * 137) % 950) as f32,
                ((i * 83) % 630) as f32,
                1.,
                1.,
            );
        }
        let a = ((now - self.received) / (1000. / 30.)).clamp(0., 1.) as f32;
        for p in &self.current.players {
            if p.lives == 0 || (p.shield && now as u32 / 100 % 2 == 0) {
                continue;
            }
            let previous = self
                .previous
                .players
                .iter()
                .find(|old| old.id == p.id)
                .unwrap_or(p);
            let x = previous.x + (p.x - previous.x) * a;
            let color = p.id.saturating_sub(1) as usize % COLORS.len();
            sprite(&mut self.batches[color], &SHIP, x, p.y, 3.);
            if p.id == self.current.you {
                rect(&mut self.batches[color], x - 19., p.y + 29., 38., 2.);
            }
        }
        for e in &self.current.enemies {
            sprite(&mut self.batches[e.kind as usize % 4], &ALIEN, e.x, e.y, 3.);
        }
        for b in &self.current.bullets {
            rect(
                &mut self.batches[if b.kind == 0 {
                    2
                } else {
                    (b.kind - 1) as usize % 8
                }],
                b.x - 2.,
                b.y,
                4.,
                13.,
            );
        }
        for (batch, color) in self.batches.iter().zip(COLORS) {
            if !batch.is_empty() {
                canvas::fill_rects(self.surface, batch, color)?;
            }
        }
        if SOCKET.get().is_none() {
            canvas::draw(
                self.surface,
                &[canvas::DrawCommand::FillText(canvas::Text {
                    value: "JOIN THE SHARED ARENA".into(),
                    at: canvas::Point { x: 205., y: 310. },
                    font: "bold 34px monospace".into(),
                    color: "#80edcf".into(),
                })],
            )?;
        }
        Ok(())
    }
    fn update(&mut self, next: Snapshot) {
        if let Some(you) = next.players.iter().find(|p| p.id == next.you) {
            let hud = format!(
                "Pilot {} · {} pilots / 8 · Wave {} · Score {} · {} lives{}",
                next.you,
                next.players.len(),
                next.wave,
                you.score,
                you.lives,
                if you.lives == 0 {
                    " — press R to respawn after 3 seconds"
                } else {
                    ""
                }
            );
            if hud != self.hud {
                set("$hud", hud.clone());
                self.hud = hud;
            }
        }
        self.previous = std::mem::replace(&mut self.current, next);
        self.received = animation::now();
    }
}
async fn receive(socket: u32) -> Result<(), String> {
    loop {
        match websocket::receive(socket).await? {
            websocket::Event::Message(websocket::Frame::Binary(bytes)) => {
                let next = Snapshot::read(&bytes)?;
                VIEW.with(|v| {
                    if let Some(view) = v.borrow_mut().as_mut() {
                        view.update(next);
                    }
                });
            }
            websocket::Event::Message(websocket::Frame::Text(text)) => {
                let message: Value =
                    serde_json::from_str(&text).map_err(|_| "Invalid arena message")?;
                if message["type"] == "welcome" {
                    set(
                        "$status",
                        format!(
                            "Connected as {} · every pilot controls their own ship",
                            message["name"].as_str().unwrap_or("pilot")
                        ),
                    );
                }
                if message["type"] == "scores" {
                    let lines: Vec<_> = message["scores"]
                        .as_array()
                        .ok_or("Invalid leaderboard")?
                        .iter()
                        .enumerate()
                        .map(|(i, p)| {
                            format!(
                                "{}. {} — {} points · wave {}",
                                i + 1,
                                p["name"].as_str().unwrap_or("Pilot"),
                                p["score"],
                                p["wave"]
                            )
                        })
                        .collect();
                    set("$scores", json!(lines));
                }
            }
            websocket::Event::Closed(info) => return Err(format!("Disconnected: {}", info.reason)),
        }
    }
}
async fn render(socket: u32) -> Result<(), String> {
    let mut heartbeat = 0.;
    loop {
        let now = animation::next_frame().await?;
        if SOCKET.get() != Some(socket) {
            return Ok(());
        }
        if now - heartbeat > 200. {
            send(input() | TAP.replace(0))?;
            heartbeat = now;
        }
        VIEW.with(|v| v.borrow_mut().as_mut().ok_or("Missing canvas")?.draw(now))?;
    }
}
fn encode(value: &str) -> String {
    value
        .bytes()
        .map(|b| {
            if b.is_ascii_alphanumeric() || b"-_.~".contains(&b) {
                (b as char).to_string()
            } else {
                format!("%{b:02X}")
            }
        })
        .collect()
}
async fn join() -> Result<(), String> {
    if BUSY.replace(true) {
        return Ok(());
    }
    set("$busy", true);
    set(
        "$status",
        "Connecting — approve the browser connection prompt",
    );
    let result = async {
        let url = get("$serverUrl");
        if !url.starts_with("ws://") && !url.starts_with("wss://") {
            return Err("Enter the deployment's ws:// or wss:// endpoint".into());
        }
        let connection = websocket::connect(
            format!(
                "{}?name={}",
                url.trim_end_matches('/'),
                encode(&get("$name"))
            ),
            vec![],
        )
        .await?;
        SOCKET.set(Some(connection.id));
        HELD.set(0);
        set("$joined", true);
        set("$busy", false);
        let mut reading = pin!(receive(connection.id));
        let mut drawing = pin!(render(connection.id));
        // Network input and browser-paced drawing share one guest task, with no SDK fork or executor dependency.
        let result = poll_fn(|cx| {
            if let std::task::Poll::Ready(value) = reading.as_mut().poll(cx) {
                return std::task::Poll::Ready(value);
            }
            drawing.as_mut().poll(cx)
        })
        .await;
        if SOCKET.get() != Some(connection.id) {
            set("$status", "Disconnected");
            return Ok(());
        }
        SOCKET.set(None);
        let _ = websocket::close(connection.id, 1000, "Leaving arena".into());
        result
    }
    .await;
    HELD.set(0);
    BUSY.set(false);
    set("$busy", false);
    set("$joined", false);
    set(
        "$hud",
        "Not connected — join the arena to get your own spaceship",
    );
    VIEW.with(|v| {
        if let Some(view) = v.borrow_mut().as_mut() {
            let _ = view.draw(animation::now());
        }
    });
    result
}
async fn action(action: &str, params: &str) -> Result<(), String> {
    let p: Value = serde_json::from_str(params).map_err(|_| "Invalid action")?;
    match action {
        "SetServer" => set("$serverUrl", p["value"].clone()),
        "SetName" => set("$name", p["value"].clone()),
        "Join" => return join().await,
        "Leave" => {
            if let Some(socket) = SOCKET.take() {
                websocket::close(socket, 1000, "Pilot left".into())?;
            }
            set("$status", "Disconnected");
        }
        "Respawn" => send(input() | 8)?,
        "Tap" => {
            let bits = p["bits"].as_u64().ok_or("Missing controls")? as u8 & 7;
            TAP.set(bits);
            send(input() | bits)?;
        }
        _ => (),
    }
    Ok(())
}
impl crate::exports::druid::openui::client::Guest for Client {
    async fn init() {
        match canvas::open("multiplayer", 960, 640) {
            Ok(surface) => {
                let mut view = View {
                    surface,
                    current: Snapshot::default(),
                    previous: Snapshot::default(),
                    received: 0.,
                    batches: std::array::from_fn(|_| Vec::with_capacity(4096)),
                    hud: String::new(),
                };
                let _ = view.draw(0.);
                VIEW.with(|v| v.replace(Some(view)));
            }
            Err(error) => set("$status", error),
        }
    }
    async fn handle_action(name: String, params: String) {
        if let Err(error) = action(&name, &params).await {
            set("$status", error);
        }
    }
}
impl crate::exports::druid::openui::keyboard::Guest for Client {
    async fn handle_event(event: crate::exports::druid::openui::keyboard::KeyEvent) {
        use crate::exports::druid::openui::keyboard::EventKind;
        if event.region != "multiplayer" {
            return;
        }
        let bit = match event.code.as_str() {
            "KeyA" => 1,
            "ArrowLeft" => 2,
            "KeyD" => 4,
            "ArrowRight" => 8,
            "Space" => 16,
            _ => 0,
        };
        match event.kind {
            EventKind::Blur => HELD.set(0),
            EventKind::Up => HELD.set(HELD.get() & !bit),
            EventKind::Down if !event.control && !event.alt && !event.meta => {
                HELD.set(HELD.get() | bit);
                if event.code == "KeyR" && !event.repeat {
                    let _ = send(input() | 8);
                }
            }
            _ => (),
        }
        if let Err(error) = send(input()) {
            set("$status", error);
        }
    }
}
