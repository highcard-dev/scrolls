#!/bin/bash
set -e

TAG=$1
echo "Tag: $TAG"

PRESIGN_OBJECT_KEY=lgsm/${TAG}-snapshot-latest.tar.gz

TMP_VOLUME_NAME=lgsm-prebuild-$(date +%s)

docker volume rm $TMP_VOLUME_NAME || true

DRUID_CONFIG=$(mktemp)

docker run --rm -v $TMP_VOLUME_NAME:/app/resources bash sh -c 'wget -O /app/resources/druid-install-command.sh https://github.com/highcard-dev/druid-cli/releases/latest/download/druid-install-command.sh && mkdir -p /app/resources/deployment && chmod +x /app/resources/druid-install-command.sh && chown 1000:1000 -R /app/resources/'

DOCKER_ENTRYPOINT=/app/resources/druid-install-command.sh

echo "Login to registry"
docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v $TMP_VOLUME_NAME:/app/resources -v $DRUID_CONFIG:$HOME/druid.yml --entrypoint /app/resources/druid-install-command.sh -w /app/resources/deployment artifacts.druid.gg/druid-team/druid:latest-nix-steamcmd registry login --host ${SCROLL_REGISTRY_HOST} --user '${SCROLL_REGISTRY_USER}' --password ${SCROLL_REGISTRY_PASSWORD}

echo "Pulling scroll"
docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v $TMP_VOLUME_NAME:/app/resources --entrypoint /app/resources/druid-install-command.sh -w /app/resources/deployment artifacts.druid.gg/druid-team/druid:latest-nix-steamcmd registry pull artifacts.druid.gg/druid-team/scroll-lgsm:${TAG}server

echo "Running scroll install script"
docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v $TMP_VOLUME_NAME:/app/resources --entrypoint /app/resources/druid-install-command.sh -w /app/resources/deployment artifacts.druid.gg/druid-team/druid:latest-nix-steamcmd run install

echo "Pushing scroll to registry"
docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v $TMP_VOLUME_NAME:/app/resources -v $DRUID_CONFIG:$HOME/druid.yml --entrypoint /app/resources/druid-install-command.sh -w /app/resources/deployment artifacts.druid.gg/druid-team/druid:latest-nix-steamcmd registry push artifacts.druid.gg/druid-team/scroll-lgsm:${TAG}server-prebuild
echo "Prebuild uploaded"

rm -f $DRUID_CONFIG
docker volume rm $TMP_VOLUME_NAME || true
