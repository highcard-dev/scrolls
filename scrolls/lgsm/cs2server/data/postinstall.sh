#!/bin/bash
set -euo pipefail

config_file="serverfiles/game/csgo/cfg/server.cfg"
password_file=".druid-rcon-password"

if [ -n "${CS2_RCON_PASSWORD:-}" ]; then
  rcon_password="$CS2_RCON_PASSWORD"
elif [ -f "$password_file" ]; then
  rcon_password="$(cat "$password_file")"
else
  rcon_password="$(tr -dc 'A-Za-z0-9' </dev/urandom | head -c 50)"
  printf '%s\n' "$rcon_password" > "$password_file"
  chmod 600 "$password_file"
fi

mkdir -p "$(dirname "$config_file")"
touch "$config_file"

if grep -q '^rcon_password ".*"' "$config_file"; then
  sed -i 's/^rcon_password ".*"/rcon_password "'"$rcon_password"'"/' "$config_file"
else
  printf '\nrcon_password "%s"\n' "$rcon_password" >> "$config_file"
fi
