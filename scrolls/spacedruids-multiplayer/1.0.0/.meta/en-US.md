---
name: SpaceDruids Multiplayer
description: Up to eight pilots, one shared alien invasion, a live WebSocket backend and persistent SQLite high scores.
---

Cooperate in a shared SpaceDruids arena. Each connected player has a separate colored spaceship and individual score. The server controls movement, bullets, hits and enemy waves; a Rust WASM client renders the game in Druid UI.

Use **arrows / A / D** to move and **Space** to fire. After losing all three lives, wait three seconds and press **R** to respawn. Click the arena to focus keyboard controls. A browser prompt asks permission before opening the game WebSocket.

High scores are stored in `highscores.sqlite` on the deployment volume and survive server restarts. This is an anonymous multiplayer demo, not a competitive service: names are not authenticated, reconnects receive a new ship, and scores are grouped by display name. The public page is listed only when the owner explicitly publishes it.
