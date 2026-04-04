#!/bin/bash
set -e

TAG=$1
echo "Tag: $TAG"

TMP_VOLUME_NAME=lgsm-prebuild-$(date +%s)

docker volume rm $TMP_VOLUME_NAME || true

docker run --rm -v $TMP_VOLUME_NAME:/app/resources bash sh -c 'wget -O /app/resources/druid-install-command.sh https://github.com/highcard-dev/druid-cli/releases/latest/download/druid-install-command.sh && mkdir -p /app/resources/deployment && chmod +x /app/resources/druid-install-command.sh && chown 1000:1000 -R /app/resources/'

echo "Pulling scroll"
docker run --rm -v $TMP_VOLUME_NAME:/app/resources --entrypoint /app/resources/druid-install-command.sh -w /app/resources/deployment artifacts.druid.gg/druid-team/druid:latest-nix-steamcmd registry pull artifacts.druid.gg/druid-team/scroll-lgsm:${TAG}server

echo "Running scroll install script"
docker run --rm -v $TMP_VOLUME_NAME:/app/resources --entrypoint /app/resources/druid-install-command.sh -w /app/resources/deployment artifacts.druid.gg/druid-team/druid:latest-nix-steamcmd run install

echo "Pushing scroll to registry"
docker run --rm -v $TMP_VOLUME_NAME:/app/resources -e DRUID_REGISTRY_HOST="${SCROLL_REGISTRY_HOST}" -e DRUID_REGISTRY_USER="${SCROLL_REGISTRY_USER}" -e DRUID_REGISTRY_PASSWORD="${SCROLL_REGISTRY_PASSWORD}" --entrypoint /app/resources/druid-install-command.sh -w /app/resources/deployment artifacts.druid.gg/druid-team/druid:latest-nix-steamcmd registry push artifacts.druid.gg/druid-team/scroll-lgsm:${TAG}server-prebuild
echo "Prebuild uploaded"

docker volume rm $TMP_VOLUME_NAME || true
