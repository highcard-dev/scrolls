# Platform Requirements for Voice Server Support

This document outlines what needs to be added to the Druid platform to support WebRTC-based voice chat in DruidUI.

## WIT Interface: `druid-ui-webrtc.wit`

The WebAssembly Interface Types (WIT) definition provides the contract between DruidUI WASM components and the host platform.

## Required Platform Features

### 1. Media Device Access

**APIs to implement:**
```
get-user-media(constraints) -> media-stream
enumerate-devices() -> list<device-info>
```

**Host responsibilities:**
- Request browser permissions for microphone access
- Enumerate available audio input/output devices
- Create MediaStream objects and pass handles to WASM
- Handle device switching/hot-plugging

**Browser APIs needed:**
- `navigator.mediaDevices.getUserMedia()`
- `navigator.mediaDevices.enumerateDevices()`

---

### 2. RTCPeerConnection

**APIs to implement:**
```
create-peer-connection(config) -> peer-connection
peer-connection.create-offer() -> session-description
peer-connection.create-answer() -> session-description
peer-connection.add-ice-candidate(candidate)
```

**Host responsibilities:**
- Create and manage RTCPeerConnection instances
- Handle SDP offer/answer negotiation
- Process ICE candidates
- Manage connection state transitions
- Relay events to WASM component

**Browser APIs needed:**
- `new RTCPeerConnection(config)`
- `createOffer()` / `createAnswer()`
- `setLocalDescription()` / `setRemoteDescription()`
- `addIceCandidate()`

---

### 3. Audio Track Control

**APIs to implement:**
```
audio-track.set-enabled(bool)  // Mute/unmute
audio-track.set-volume(f32)    // Volume control
audio-track.get-audio-level()  // VU meter
```

**Host responsibilities:**
- Enable/disable audio tracks
- Apply volume transformations
- Calculate audio levels for visualization
- Handle audio routing (which speakers, etc.)

**Browser APIs needed:**
- `MediaStreamTrack.enabled`
- `GainNode` for volume control (Web Audio API)
- `AnalyserNode` for audio level detection

---

### 4. WebSocket Signaling

**APIs to implement:**
```
websocket-connect(url) -> websocket
websocket.send-text(string)
websocket.send-binary(bytes)
```

**Host responsibilities:**
- Establish WebSocket connections
- Send/receive signaling messages
- Handle connection lifecycle
- Relay messages to WASM component

**Browser APIs needed:**
- `new WebSocket(url)`
- `send()` for text/binary
- Event handlers: `onopen`, `onmessage`, `onerror`, `onclose`

---

### 5. Event System

**Event flow:**
```
Browser Event → Host Platform → WASM Component
    ↓                ↓                ↓
ICE Candidate → webrtc-event → handle-webrtc-event()
Remote Track  → track-added  → render update
WS Message    → ws-event     → handle-ws-event()
```

**Host responsibilities:**
- Register event listeners on browser APIs
- Serialize events into WIT-compatible format
- Call exported WASM event handlers
- Manage event queue/batching for performance

---

## Implementation Priority

### Phase 1: Minimum Viable (voice-only, no UI controls)
1. `get-user-media` (audio only)
2. `create-peer-connection`
3. SDP offer/answer
4. ICE candidates
5. WebSocket signaling

**Deliverable:** Basic voice connection between two peers

---

### Phase 2: Audio Control
1. Mute/unmute
2. Audio level detection
3. Volume control
4. Device enumeration

**Deliverable:** User can mute, adjust volume, see who's talking

---

### Phase 3: Advanced Features
1. Multi-peer connections (SFU integration)
2. Screen sharing (video tracks)
3. Recording capabilities
4. Network quality metrics

**Deliverable:** Full-featured voice chat

---

## Security Considerations

### Permissions
- All media device access requires explicit user permission
- WASM components cannot bypass browser security model
- Host must enforce permission checks before granting access

### Sandboxing
- WASM components get **handles** to resources, not direct access
- Host owns all WebRTC objects (PeerConnection, MediaStream)
- Component can only invoke operations via WIT interface
- No access to raw UDP/TCP sockets

### Data Privacy
- Audio data flows through browser's WebRTC stack
- WASM component sees only control plane (SDP, ICE)
- Media encryption (DTLS/SRTP) handled by browser

---

## Testing Requirements

### Browser Compatibility
Test on:
- Chrome/Chromium (primary target)
- Firefox
- Safari (macOS/iOS)
- Edge

### Functionality Tests
1. Microphone permission flow
2. Peer connection establishment
3. Audio transmission (can hear remote peer)
4. Mute/unmute functionality
5. WebSocket reconnection
6. Multiple simultaneous connections
7. Network interruption handling

### Performance Tests
- Latency measurements (mouth-to-ear delay)
- Audio quality under packet loss
- CPU usage in WASM component
- Memory usage with N peers

---

## Integration with Existing Scroll

Once platform features are ready, the mediasoup scroll can be updated:

### Backend (already complete)
- ✅ mediasoup server.js
- ✅ WebSocket signaling
- ✅ SFU multi-party logic

### Frontend (needs platform APIs)
- ⏳ Rewrite `.scroll/private/` using actual DruidUI components
- ⏳ Use WIT interfaces instead of plain browser APIs
- ⏳ WASM component for voice UI

### Example DruidUI Component (pseudo-code)
```typescript
// voice-client.component.ts (compiled to WASM)
import { getUserMedia, createPeerConnection } from 'druid:webrtc';

export function init() {
  // Called when component loads
}

export function render(): string {
  // Return component HTML
  return `<div class="voice-panel">...</div>`;
}

export function onUserAction(action: string, data: string) {
  if (action === 'join') {
    const stream = getUserMedia({ audio: { echoCancellation: true } });
    const pc = createPeerConnection({ iceServers: [...] });
    pc.addStream(stream);
    // ...
  }
}

export function handleWebrtcEvent(id: number, event: WebRTCEvent) {
  if (event.kind === 'ice-candidate') {
    // Send to signaling server via WebSocket
  }
}
```

---

## Questions for Platform Team

1. **Does DruidUI already expose any browser APIs to WASM?**
   - If yes: which ones? Can we extend the pattern?
   - If no: what's the preferred architecture for host-WASM communication?

2. **Resource limits for WASM components?**
   - Max number of peer connections per component?
   - Memory limits?
   - CPU time limits?

3. **Event handling model?**
   - Synchronous or async callbacks?
   - Event batching/throttling?
   - Priority system for real-time events?

4. **Multi-component coordination?**
   - Can two DruidUI components share a peer connection?
   - How do components communicate with the server backend?

---

## Timeline Estimate

**Assuming 1 developer working full-time:**

- **Phase 1 (MVP):** 2-3 weeks
  - WIT implementation in platform: 1 week
  - WASM bindings generation: 2-3 days
  - Testing/debugging: 1 week

- **Phase 2 (Audio control):** 1 week
  - Additional APIs: 2-3 days
  - UI components: 2-3 days
  - Integration testing: 1-2 days

- **Phase 3 (Advanced):** 2+ weeks
  - Feature-dependent

**Total for production-ready voice chat:** 4-6 weeks

---

## Related Work

### Existing WebRTC WASM projects to study:
- **webrtc-rs** (Rust WebRTC implementation)
- **Jitsi Meet** (WebRTC video conferencing)
- **mediasoup-client** (TypeScript WebRTC client)

### Component Model resources:
- https://component-model.bytecodealliance.org/
- https://github.com/WebAssembly/component-model

---

**Author:** Lugh (Druid Bot)  
**Date:** 2026-02-11  
**For:** mediasoup voice server scroll PR #13
