#!/bin/bash
set -e

MAX=${DRUID_MAX_MEMORY:-}
MAX=${MAX%?}
if [ -z "${MAX}" ];
then
    MAX=1024M
fi

if [ ! -f server.properties ] && [ -f server.properties.scroll_template ]; then
    if [ -f .druid-rcon-password ]; then
        RCON_PASSWORD=$(cat .druid-rcon-password)
    else
        umask 077
        RCON_PASSWORD=$(LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 50)
        printf '%s\n' "$RCON_PASSWORD" > .druid-rcon-password
    fi
    sed "s/{{ .Config.rcon.password }}/$RCON_PASSWORD/g" server.properties.scroll_template > server.properties
fi

java -Xmx$MAX -Xms1024M -jar spigot.jar nogui
