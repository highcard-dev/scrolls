#!/bin/bash
set -e
curl -O "https://raw.githubusercontent.com/GameServerManagers/LinuxGSM/master/lgsm/data/serverlist.csv"

#optional shortname to build only one image
wanted_image=$1

while read line; do
  export shortname=$(echo "$line" | awk -F, '{ print $1 }')
  export servername=$(echo "$line" | awk -F, '{ print $2 }')
  export gamename=$(echo "$line" | awk -F, '{ print $3 }')
  export distro=$(echo "$line" | awk -F, '{ print $4 }')
  
  if [ -n "$wanted_image" ] && [ "$wanted_image" != "$shortname" ]; then
    continue
  fi

  echo "Generating Dockerfile.${shortname} (${gamename})"
  echo "shortname ${shortname}"
  docker build -f Dockerfile . --build-arg SHORTNAME="${shortname}" --build-arg SERVERNAME="${servername}" --build-arg GAMENAME="${gamename}" --build-arg DISTRO="${distro}" -t "highcard/druidd-lgsm:${shortname}" --no-cache
  echo "Pushing highcard/druidd-lgsm:${shortname}"
  docker push "highcard/druidd-lgsm:${shortname}"
done < <(tail -n +2 serverlist.csv)

rm serverlist.csv