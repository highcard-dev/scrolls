#!/bin/sh
set -eu

export LD_LIBRARY_PATH="${LD_LIBRARY_PATH:-}:$(pwd)"

rcon_password="${RCON_PASSWORD:-${DRUID_RCON_PASSWORD:-}}"
if [ -z "${rcon_password}" ]; then
  if [ -f .rcon-password ]; then
    rcon_password="$(cat .rcon-password)"
  else
    rcon_password="$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')"
    printf '%s' "${rcon_password}" > .rcon-password
    chmod 600 .rcon-password
  fi
fi

exec ./RustDedicated -batchmode -nographics \
   -app.port "${DRUID_PORT_RUSTPLUS_1:-28082}" \
   -app.publicip "${DRUID_IP_1:-0.0.0.0}" \
   -server.ip "0.0.0.0" \
   -server.port "${DRUID_PORT_MAIN_1:-28015}" \
   -server.queryport "${DRUID_PORT_QUERY_1:-28017}" \
   -rcon.ip "0.0.0.0" \
   -rcon.port "${DRUID_PORT_RCON_1:-28016}" \
   -rcon.password "${rcon_password}" \
   -server.maxplayers 75 \
   -server.hostname "Rust Oxide Server by druid.gg" \
   -server.identity "druid" \
   -server.level "Procedural Map" \
   -server.worldsize 1000 \
   -server.saveinterval 300 \
   -server.globalchat true \
   -server.description "A Server hosted on druid.gg" \
   -server.headerimage "https://druid.gg/" \
   -server.url "https://druid.gg/"
