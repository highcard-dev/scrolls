
if [ ! -f rcon_password ]; then
   openssl rand -hex 20 > rcon_password
fi

DRUID_PASSWORD_RCON=$(cat rcon_password)

./RustDedicated -batchmode -nographics \
   -app.port $DRUID_PORT_RUSTPLUS_1 \
   -server.ip "0.0.0.0" \
   -server.port 28015 \
   -rcon.ip "0.0.0.0" \
   -rcon.port $DRUID_PORT_RCON_1 \
   -rcon.password $DRUID_PASSWORD_RCON \
   -server.maxplayers 75 \
   -server.hostname "Druid Test Server" \
   -server.identity "my_server_identity" \
   -server.level "Procedural Map" \
   -server.seed 12345 \
   -server.worldsize 3000 \
   -server.saveinterval 300 \
   -server.globalchat true \
   -server.description "A druid test server" \
   -server.headerimage "https://druid.gg/" \
   -server.url "https://druid.gg/"