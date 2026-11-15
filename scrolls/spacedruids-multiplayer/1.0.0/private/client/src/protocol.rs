#[derive(Clone, Default)]
pub struct Player {
    pub id: u8,
    pub lives: u8,
    pub shield: bool,
    pub x: f32,
    pub y: f32,
    pub score: u32,
}
#[derive(Clone, Default)]
pub struct Dot {
    pub x: f32,
    pub y: f32,
    pub kind: u8,
}
#[derive(Default)]
pub struct Snapshot {
    pub you: u8,
    pub wave: u16,
    pub players: Vec<Player>,
    pub enemies: Vec<Dot>,
    pub bullets: Vec<Dot>,
}
impl Snapshot {
    pub fn read(bytes: &[u8]) -> Result<Self, String> {
        if bytes.len() < 12 || bytes[0] != 1 {
            return Err("Unsupported arena snapshot".into());
        }
        let u16at = |at| u16::from_le_bytes([bytes[at], bytes[at + 1]]);
        let (players, enemies, bullets) =
            (bytes[8] as usize, bytes[9] as usize, u16at(10) as usize);
        if players > 8
            || enemies > 40
            || bullets > 256
            || bytes.len() != 12 + players * 12 + (enemies + bullets) * 6
        {
            return Err("Invalid arena snapshot size".into());
        }
        let mut result = Self {
            you: bytes[1],
            wave: u16at(2),
            ..Self::default()
        };
        let mut at = 12;
        for _ in 0..players {
            result.players.push(Player {
                id: bytes[at],
                lives: bytes[at + 1],
                shield: bytes[at + 2] != 0,
                x: u16at(at + 4) as f32,
                y: u16at(at + 6) as f32,
                score: u32::from_le_bytes(bytes[at + 8..at + 12].try_into().unwrap()),
            });
            at += 12;
        }
        for _ in 0..enemies {
            result.enemies.push(Dot {
                x: u16at(at) as f32,
                y: u16at(at + 2) as f32,
                kind: bytes[at + 4],
            });
            at += 6;
        }
        for _ in 0..bullets {
            result.bullets.push(Dot {
                x: u16at(at) as f32,
                y: u16at(at + 2) as f32,
                kind: bytes[at + 4],
            });
            at += 6;
        }
        Ok(result)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn reads_a_binary_ship_and_rejects_truncated_frames() {
        let mut bytes = vec![0u8; 24];
        bytes[0] = 1;
        bytes[1] = 2;
        bytes[2] = 3;
        bytes[8] = 1;
        bytes[12] = 2;
        bytes[13] = 3;
        bytes[16..18].copy_from_slice(&480u16.to_le_bytes());
        bytes[20..24].copy_from_slice(&1234u32.to_le_bytes());
        let state = Snapshot::read(&bytes).unwrap();
        assert_eq!(state.you, 2);
        assert_eq!(state.wave, 3);
        assert_eq!(state.players[0].x, 480.);
        assert_eq!(state.players[0].score, 1234);
        assert!(Snapshot::read(&bytes[..23]).is_err());
        bytes[8] = 9;
        assert!(Snapshot::read(&bytes).is_err());
    }
    #[test]
    fn reads_empty_enemies_and_an_enemy_projectile() {
        let mut bytes = vec![0; 18];
        bytes[0] = 1;
        bytes[10] = 1;
        bytes[12..14].copy_from_slice(&100u16.to_le_bytes());
        let state = Snapshot::read(&bytes).unwrap();
        assert!(state.enemies.is_empty());
        assert_eq!(state.bullets[0].x, 100.);
    }
}
