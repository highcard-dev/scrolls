#!/usr/bin/env node
/**
 * Druid Voice Server - mediasoup WebRTC SFU
 * Browser-based voice chat for gaming communities
 */

const express = require('express');
const http = require('http');
const WebSocket = require('ws');
const mediasoup = require('mediasoup');

const HTTP_PORT = process.env.PORT_HTTP || 3000;
const WS_PORT = process.env.PORT_HTTP || 3000;
const WEBRTC_MIN_PORT = parseInt(process.env.PORT_WEBRTC_MIN) || 40000;
const WEBRTC_MAX_PORT = parseInt(process.env.PORT_WEBRTC_MAX) || 40100;

const app = express();
const server = http.createServer(app);
const wss = new WebSocket.Server({ server });

let worker;
let router;
const transports = new Map();
const producers = new Map();
const consumers = new Map();

// mediasoup worker settings
const mediaCodecs = [
  {
    kind: 'audio',
    mimeType: 'audio/opus',
    clockRate: 48000,
    channels: 2
  }
];

async function createWorker() {
  worker = await mediasoup.createWorker({
    logLevel: 'warn',
    rtcMinPort: WEBRTC_MIN_PORT,
    rtcMaxPort: WEBRTC_MAX_PORT,
  });

  console.log(`✅ mediasoup worker created [pid:${worker.pid}]`);

  worker.on('died', () => {
    console.error('❌ mediasoup worker died, exiting...');
    process.exit(1);
  });

  router = await worker.createRouter({ mediaCodecs });
  console.log('✅ mediasoup router created');
}

// WebSocket signaling
wss.on('connection', (ws) => {
  console.log('🔌 Client connected');

  ws.on('message', async (message) => {
    try {
      const data = JSON.parse(message);
      
      switch (data.type) {
        case 'getRouterRtpCapabilities':
          ws.send(JSON.stringify({
            type: 'routerRtpCapabilities',
            data: router.rtpCapabilities
          }));
          break;

        case 'createWebRtcTransport':
          const transport = await createWebRtcTransport();
          transports.set(transport.id, transport);
          ws.send(JSON.stringify({
            type: 'transportCreated',
            data: {
              id: transport.id,
              iceParameters: transport.iceParameters,
              iceCandidates: transport.iceCandidates,
              dtlsParameters: transport.dtlsParameters,
            }
          }));
          break;

        case 'connectTransport':
          const t = transports.get(data.transportId);
          await t.connect({ dtlsParameters: data.dtlsParameters });
          ws.send(JSON.stringify({ type: 'transportConnected' }));
          break;

        case 'produce':
          const producer = await transports.get(data.transportId).produce({
            kind: data.kind,
            rtpParameters: data.rtpParameters,
          });
          producers.set(producer.id, producer);
          ws.send(JSON.stringify({
            type: 'produced',
            data: { id: producer.id }
          }));
          break;

        case 'consume':
          const consumer = await transports.get(data.transportId).consume({
            producerId: data.producerId,
            rtpCapabilities: data.rtpCapabilities,
            paused: true,
          });
          consumers.set(consumer.id, consumer);
          ws.send(JSON.stringify({
            type: 'consumed',
            data: {
              id: consumer.id,
              producerId: data.producerId,
              kind: consumer.kind,
              rtpParameters: consumer.rtpParameters,
            }
          }));
          break;
      }
    } catch (err) {
      console.error('❌ Error handling message:', err);
      ws.send(JSON.stringify({ type: 'error', error: err.message }));
    }
  });

  ws.on('close', () => {
    console.log('🔌 Client disconnected');
  });
});

async function createWebRtcTransport() {
  const transport = await router.createWebRtcTransport({
    listenIps: [{ ip: '0.0.0.0', announcedIp: process.env.ANNOUNCED_IP || null }],
    enableUdp: true,
    enableTcp: true,
    preferUdp: true,
  });
  
  return transport;
}

// HTTP API
app.get('/health', (req, res) => {
  res.json({ 
    status: 'ok',
    worker: worker?.pid,
    transports: transports.size,
    producers: producers.size,
    consumers: consumers.size
  });
});

app.get('/', (req, res) => {
  res.send(`
    <html>
      <head><title>Druid Voice Server</title></head>
      <body>
        <h1>🎙️ Druid Voice Server</h1>
        <p>Powered by mediasoup WebRTC SFU</p>
        <p>Status: <strong>Running</strong></p>
        <p>Connect via DruidUI to join voice channels</p>
      </body>
    </html>
  `);
});

// Start server
async function main() {
  await createWorker();
  
  server.listen(HTTP_PORT, () => {
    console.log(`🎙️  Druid Voice Server running`);
    console.log(`📡 HTTP/WS: http://0.0.0.0:${HTTP_PORT}`);
    console.log(`🔊 WebRTC ports: ${WEBRTC_MIN_PORT}-${WEBRTC_MAX_PORT}`);
  });
}

// Graceful shutdown
process.on('SIGTERM', async () => {
  console.log('📴 Shutting down gracefully...');
  server.close();
  worker?.close();
  process.exit(0);
});

main().catch(err => {
  console.error('❌ Failed to start server:', err);
  process.exit(1);
});
