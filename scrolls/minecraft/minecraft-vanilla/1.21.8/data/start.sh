#!/bin/bash
set -e

MAX=${DRUID_MAX_MEMORY:-}
MAX=${MAX%?}
if [ -z "${MAX}" ];
then
    MAX=1024M
fi

if [ ! -f server.properties ] && [ -f server.properties.default ]; then
    if [ -f .druid-rcon-password ]; then
        RCON_PASSWORD=$(cat .druid-rcon-password)
    else
        umask 077
        RCON_PASSWORD=$(LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 50 || true)
        if [ "${#RCON_PASSWORD}" -ne 50 ]; then
            echo "Failed to generate RCON password" >&2
            exit 1
        fi
        printf '%s\n' "$RCON_PASSWORD" > .druid-rcon-password
    fi
    sed \
        -e "s|__DRUID_RCON_PASSWORD__|$RCON_PASSWORD|g" \
        -e "s|__DRUID_PORT_MAIN__|${DRUID_PORT_MAIN_1:-25565}|g" \
        -e "s|__DRUID_PORT_RCON__|${DRUID_PORT_RCON_1:-25575}|g" \
        server.properties.default > server.properties
fi

java -Xmx$MAX -Xms1024M -jar server.jar nogui
