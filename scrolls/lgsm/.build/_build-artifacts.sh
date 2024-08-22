#!/bin/bash

curl -O "https://raw.githubusercontent.com/GameServerManagers/LinuxGSM/master/lgsm/data/serverlist.csv"

echo "{" > artifacts.json

while read line; do
  export shortname=$(echo "$line" | awk -F, '{ print $1 }')
  export servername=$(echo "$line" | awk -F, '{ print $2 }')
  export gamename=$(echo "$line" | awk -F, '{ print $3 }')
  export distro=$(echo "$line" | awk -F, '{ print $4 }')
  echo "Generating ${shortname} ${servername} (${gamename})"
  echo "\"${servername}\":\"artifacts.druid.gg/druid-team/scroll-lgsm:${servername}\"," >> artifacts.json
done < <(tail -n +2 serverlist.csv)
truncate -s -2 artifacts.json
echo "}" >> artifacts.json

rm serverlist.csv