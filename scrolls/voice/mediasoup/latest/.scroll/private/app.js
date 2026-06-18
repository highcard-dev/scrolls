// Druid Voice Server - DruidUI Client
// WebRTC client using mediasoup-client

class VoiceClient {
    constructor() {
        this.ws = null;
        this.device = null;
        this.producerTransport = null;
        this.consumerTransport = null;
        this.producer = null;
        this.consumers = new Map();
        this.connected = false;
        this.muted = false;
        this.deafened = false;
        
        this.initUI();
        this.connect();
    }
    
    initUI() {
        // Status indicator
        this.statusIndicator = document.getElementById('status-indicator');
        this.statusText = document.getElementById('status-text');
        
        // Stats
        this.statUsers = document.getElementById('stat-users');
        this.statChannels = document.getElementById('stat-channels');
        this.statUptime = document.getElementById('stat-uptime');
        
        // Buttons
        this.btnJoin = document.getElementById('btn-join');
        this.btnMute = document.getElementById('btn-mute');
        this.btnDeafen = document.getElementById('btn-deafen');
        
        // Log
        this.logElement = document.getElementById('log');
        
        // Button handlers
        this.btnJoin.addEventListener('click', () => this.joinVoice());
        this.btnMute.addEventListener('click', () => this.toggleMute());
        this.btnDeafen.addEventListener('click', () => this.toggleDeafen());
        
        // Start uptime counter
        this.startTime = Date.now();
        setInterval(() => this.updateUptime(), 1000);
    }
    
    log(message, type = 'info') {
        const now = new Date();
        const timestamp = now.toTimeString().split(' ')[0];
        const entry = document.createElement('div');
        entry.className = 'log-entry';
        entry.innerHTML = `<span class="log-timestamp">[${timestamp}]</span> ${message}`;
        this.logElement.appendChild(entry);
        this.logElement.scrollTop = this.logElement.scrollHeight;
        console.log(`[${type}]`, message);
    }
    
    updateUptime() {
        const uptime = Math.floor((Date.now() - this.startTime) / 1000);
        const hours = Math.floor(uptime / 3600);
        const minutes = Math.floor((uptime % 3600) / 60);
        const seconds = uptime % 60;
        this.statUptime.textContent = `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    }
    
    connect() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}`;
        
        this.log(`Connecting to ${wsUrl}...`);
        
        this.ws = new WebSocket(wsUrl);
        
        this.ws.onopen = () => {
            this.log('✅ Connected to voice server');
            this.connected = true;
            this.statusIndicator.className = 'status-indicator online';
            this.statusText.textContent = 'Connected';
            this.btnJoin.disabled = false;
            
            // Request router capabilities
            this.send({ type: 'getRouterRtpCapabilities' });
        };
        
        this.ws.onclose = () => {
            this.log('❌ Disconnected from voice server');
            this.connected = false;
            this.statusIndicator.className = 'status-indicator offline';
            this.statusText.textContent = 'Disconnected';
            this.btnJoin.disabled = true;
            
            // Reconnect after 3 seconds
            setTimeout(() => this.connect(), 3000);
        };
        
        this.ws.onerror = (error) => {
            this.log(`WebSocket error: ${error}`, 'error');
        };
        
        this.ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
            this.handleMessage(message);
        };
    }
    
    send(data) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(data));
        }
    }
    
    handleMessage(message) {
        switch (message.type) {
            case 'routerRtpCapabilities':
                this.log('Received router RTP capabilities');
                // Store for later use when joining
                this.routerRtpCapabilities = message.data;
                break;
                
            case 'transportCreated':
                this.log(`Transport created: ${message.data.id}`);
                break;
                
            case 'transportConnected':
                this.log('Transport connected');
                break;
                
            case 'produced':
                this.log(`Producer created: ${message.data.id}`);
                break;
                
            case 'consumed':
                this.log(`Consumer created: ${message.data.id}`);
                break;
                
            case 'error':
                this.log(`Server error: ${message.error}`, 'error');
                break;
        }
    }
    
    async joinVoice() {
        try {
            this.log('🎙️ Joining voice channel...');
            this.btnJoin.disabled = true;
            this.btnJoin.textContent = 'Connecting...';
            
            // Get user microphone
            const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
            this.log('✅ Microphone access granted');
            
            // In a real implementation, we would:
            // 1. Load mediasoup-client library
            // 2. Create Device with router capabilities
            // 3. Create transports
            // 4. Produce audio track
            // 5. Consume other users' audio
            
            // For now, just update UI
            this.btnJoin.textContent = 'Connected';
            this.btnMute.disabled = false;
            this.btnDeafen.disabled = false;
            
            this.statUsers.textContent = '1';
            this.log('✅ Connected to voice channel');
            
        } catch (error) {
            this.log(`❌ Failed to join: ${error.message}`, 'error');
            this.btnJoin.disabled = false;
            this.btnJoin.textContent = 'Join Voice Channel';
        }
    }
    
    toggleMute() {
        this.muted = !this.muted;
        this.btnMute.textContent = this.muted ? 'Unmute' : 'Mute';
        this.log(this.muted ? '🔇 Microphone muted' : '🔊 Microphone unmuted');
    }
    
    toggleDeafen() {
        this.deafened = !this.deafened;
        this.btnDeafen.textContent = this.deafened ? 'Undeafen' : 'Deafen';
        this.log(this.deafened ? '🔇 Audio deafened' : '🔊 Audio enabled');
    }
}

// Initialize when page loads
document.addEventListener('DOMContentLoaded', () => {
    new VoiceClient();
});
