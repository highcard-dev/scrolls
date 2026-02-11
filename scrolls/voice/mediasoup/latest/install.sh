#!/bin/bash
set -e

cd /app/resources/deployment

# Initialize package.json if it doesn't exist
if [ ! -f "package.json" ]; then
  cat > package.json << 'EOF'
{
  "name": "druid-voice-mediasoup",
  "version": "1.0.0",
  "description": "Druid Voice Server powered by mediasoup",
  "main": "server.js",
  "dependencies": {
    "mediasoup": "^3.14.0",
    "express": "^4.18.2",
    "ws": "^8.16.0"
  }
}
EOF
fi

# Install dependencies
yarn install

echo "✅ mediasoup voice server installed successfully"
