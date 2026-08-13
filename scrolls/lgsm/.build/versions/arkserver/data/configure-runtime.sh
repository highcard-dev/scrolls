#!/bin/sh
set -eu

main_port="${DRUID_PORT_MAIN_1:?DRUID_PORT_MAIN_1 is required}"
query_port="${DRUID_PORT_QUERY_1:?DRUID_PORT_QUERY_1 is required}"
rcon_port="${DRUID_PORT_RCON_1:?DRUID_PORT_RCON_1 is required}"

validate_port() {
  name="$1"
  value="$2"
  case "$value" in
    ''|*[!0-9]*)
      echo "$name must be a numeric port, got: $value" >&2
      exit 1
      ;;
  esac
  if [ "$value" -lt 1 ] || [ "$value" -gt 65535 ]; then
    echo "$name must be between 1 and 65535, got: $value" >&2
    exit 1
  fi
}

validate_port DRUID_PORT_MAIN_1 "$main_port"
validate_port DRUID_PORT_QUERY_1 "$query_port"
validate_port DRUID_PORT_RCON_1 "$rcon_port"

password_file=".druid-rcon-password"
if [ -n "${ARK_RCON_PASSWORD:-}" ]; then
  rcon_password="$ARK_RCON_PASSWORD"
elif [ -f "$password_file" ]; then
  rcon_password="$(cat "$password_file")"
else
  rcon_password="$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 50 || true)"
fi

case "$rcon_password" in
  ''|*[!A-Za-z0-9_-]*)
    echo "ARK_RCON_PASSWORD must contain only letters, numbers, underscores, or hyphens" >&2
    exit 1
    ;;
esac

umask 077
printf '%s\n' "$rcon_password" > "$password_file"
chmod 600 "$password_file"
umask 022

instance_config="lgsm/config-lgsm/arkserver/arkserver.cfg"
common_config="lgsm/config-lgsm/arkserver/common.cfg"
mkdir -p "$(dirname "$instance_config")"
{
  printf 'port="%s"\n' "$main_port"
  printf 'queryport="%s"\n' "$query_port"
  printf 'rconport="%s"\n' "$rcon_port"
  if [ -n "${DRUID_IP:-}" ]; then
    printf 'publicip="%s"\n' "$DRUID_IP"
  fi
} > "$instance_config"

# LGSM replaces publicip with its cached egress address after loading config.
if [ -n "${DRUID_IP:-}" ]; then
  mkdir -p lgsm/tmp
  printf '{"ip":"%s","country":"","countryCode":"","apiurl":"druid"}\n' \
    "$DRUID_IP" > lgsm/tmp/publicip.json
fi

{
  printf 'RCONEnabled=True\n'
  printf 'RCONPort=%s\n' "$rcon_port"
} > "$common_config"

game_config="serverfiles/ShooterGame/Saved/Config/LinuxServer/GameUserSettings.ini"
default_config="${game_config}.default"
mkdir -p "$(dirname "$game_config")"
if [ ! -f "$game_config" ]; then
  if [ -f "$default_config" ]; then
    cp "$default_config" "$game_config"
  else
    : > "$game_config"
  fi
fi

tmp_config="${game_config}.tmp"
awk -v rcon_port="$rcon_port" -v rcon_password="$rcon_password" '
function emit_missing() {
  if (!seen_port) print "RCONPort=" rcon_port
  if (!seen_enabled) print "RCONEnabled=True"
  if (!seen_password) print "ServerAdminPassword=" rcon_password
}
BEGIN {
  in_server_settings = 0
  found_server_settings = 0
  seen_port = 0
  seen_enabled = 0
  seen_password = 0
}
$0 == "[ServerSettings]" {
  in_server_settings = 1
  found_server_settings = 1
  print
  next
}
/^\[/ {
  if (in_server_settings) emit_missing()
  in_server_settings = 0
  print
  next
}
in_server_settings && /^RCONPort=/ {
  if (!seen_port) print "RCONPort=" rcon_port
  seen_port = 1
  next
}
in_server_settings && /^RCONEnabled=/ {
  if (!seen_enabled) print "RCONEnabled=True"
  seen_enabled = 1
  next
}
in_server_settings && /^ServerAdminPassword=/ {
  if (!seen_password) print "ServerAdminPassword=" rcon_password
  seen_password = 1
  next
}
{ print }
END {
  if (in_server_settings) {
    emit_missing()
  } else if (!found_server_settings) {
    print ""
    print "[ServerSettings]"
    print "RCONPort=" rcon_port
    print "RCONEnabled=True"
    print "ServerAdminPassword=" rcon_password
  }
}
' "$game_config" > "$tmp_config"
mv "$tmp_config" "$game_config"
