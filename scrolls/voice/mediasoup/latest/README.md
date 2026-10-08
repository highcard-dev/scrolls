# Druid Voice Server - mediasoup Scroll

Browser-based voice chat server powered by mediasoup WebRTC SFU.

## Features

- ✅ **Browser-Based** - No client downloads required
- ✅ **Low Latency** - WebRTC for real-time voice
- ✅ **Self-Hosted** - Full control over your infrastructure
- ✅ **ColdStarter Ready** - Auto-wake on connection
- ✅ **DruidUI** - Native web management interface
- ✅ **Pay-Per-Second** - Druid's usage-based billing

## Perfect For

- Gaming communities wanting Discord alternatives
- Privacy-conscious groups
- Teams needing self-hosted voice
- Communities affected by Discord's photo ID requirements

## Technical Stack

- **Backend:** mediasoup (WebRTC SFU)
- **Runtime:** Node.js 22
- **Frontend:** DruidUI (WASM)
- **Protocol:** WebRTC over UDP + TCP

## Ports

- **HTTP/WS:** TCP port for API and WebSocket signaling
- **WebRTC:** UDP port range for media streams
  - Auto-wakes on STUN/DTLS packets
  - ColdStarter detects ICE connectivity checks

## How It Works

### ColdStarter Integration

The scroll includes a custom Lua packet handler (`webrtc.lua`) that detects:
- **STUN packets** - ICE connectivity checks
- **DTLS packets** - WebRTC handshakes

When a user attempts to join, ColdStarter automatically wakes the server.

### Architecture

```
User Browser (WebRTC)
    ↓
WebSocket Signaling (port HTTP)
    ↓
mediasoup SFU (server.js)
    ↓
UDP Media Streams (ports WEBRTC-MIN to WEBRTC-MAX)
```

## Usage

1. Deploy the scroll via Druid
2. Access DruidUI at `https://your-server:port`
3. Click "Join Voice Channel" to connect
4. Browser will request microphone permissions
5. Start talking!

## Environment Variables

Set in Druid deployment:

- `ANNOUNCED_IP` - Public IP for WebRTC (optional, auto-detected)

## Development

### Local Testing

```bash
cd /app/resources/deployment
yarn install
node server.js
```

### Files

- `scroll.yaml` - Scroll configuration
- `server.js` - mediasoup backend
- `install.sh` - Dependency installer
- `packet_handler/webrtc.lua` - ColdStarter wake logic
- `.scroll/private/` - DruidUI interface

## Cost Efficiency

With Druid's pay-per-second model:

- **Idle:** €0/month (server sleeps)
- **Active (1h/day):** ~€1-2/month
- **Active (24/7):** ~€10-15/month

Compare to Discord Nitro: €10/month with no control.

## Privacy Benefits

- ✅ Self-hosted (you own the data)
- ✅ No photo ID requirements
- ✅ No corporate surveillance
- ✅ E2E encryption capable (DTLS)
- ✅ Full GDPR compliance

## Roadmap

- [ ] Screen sharing support
- [ ] Multiple voice channels
- [ ] User permissions system
- [ ] Recording capabilities
- [ ] Mobile browser optimization
- [ ] Integration with Matrix/Element

## Related Scrolls

- Discord alternatives: Matrix, Rocket.Chat (coming soon)
- Game servers: Minecraft, Rust, etc.

## License

Part of the Druid scroll collection.

---

**Built by:** druid.gg team  
**Support:** https://discord.gg/druid
