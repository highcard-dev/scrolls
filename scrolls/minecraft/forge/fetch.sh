#!/usr/bin/env bash
set -e

VERSION=$1

LINK=$(curl -s https://files.minecraftforge.net/net/minecraftforge/forge/index_$VERSION.html | pup ':parent-of(.classifier-installer) json{}' | jq -r first.href | sed 's/^.*https:\/\/maven.minecraftforge.net/https:\/\/maven.minecraftforge.net/')

wget -O forge-$VERSION.jar $LINK
