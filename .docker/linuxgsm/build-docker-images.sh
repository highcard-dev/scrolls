#!/bin/bash

curl -O "https://raw.githubusercontent.com/GameServerManagers/LinuxGSM/master/lgsm/data/serverlist.csv"

while read line; do
  export shortname=$(echo "$line" | awk -F, '{ print $1 }')
  export servername=$(echo "$line" | awk -F, '{ print $2 }')
  export gamename=$(echo "$line" | awk -F, '{ print $3 }')
  export distro=$(echo "$line" | awk -F, '{ print $4 }')
  echo "Generating Dockerfile.${shortname} (${gamename})"
  echo "shortname ${shortname}"
  docker build -f Dockerfile . --build-arg SHORTNAME="${shortname}" --build-arg SERVERNAME="${servername}" --build-arg GAMENAME="${gamename}" --build-arg DISTRO="${distro}" -t "highcard/gsm:${shortname}"
done < <(tail -n +2 serverlist.csv)

rm serverlist.csv