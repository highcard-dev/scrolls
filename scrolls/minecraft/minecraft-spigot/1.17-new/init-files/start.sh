#!/bin/bash

if [ ! -f .rcon_password ]; then
   openssl rand -hex 20 > .rcon_password
fi

DRUID_PASSWORD_RCON=$(cat .rcon_password)

MAX=${DRUID_MAX_MEMORY%?}
if [ -z "${MAX}" ];
then
    MAX=1024M
fi

java -Xmx$MAX -Xms1024M -jar spigot.jar nogui