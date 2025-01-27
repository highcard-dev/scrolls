#!/bin/bash
set -e

TAG=$1
echo "Tag: $TAG"

export PRESIGN_OBJECT_KEY=lgsm/${TAG}-snapshot-latest.tar.gz

PRESIGNED_URL=$(cd scripts/presign/ && go run main.go)


docker run --rm -v build-folder:/app/resources bash sh -c 'wget -O /app/resources/druid-install-command.sh https://github.com/highcard-dev/druid-cli/releases/latest/download/druid-install-command.sh && mkdir /app/resources/deployment && chmod +x /app/resources/druid-install-command.sh'

DOCKER_ENTRYPOINT=/app/resources/druid-install-command.sh

echo "Running docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v build-folder:/app/resources --entrypoint /app/resources/druid-install-command.sh -w /app/resources/deployment artifacts.druid.gg/druid-team/druidd-lgsm:$TAG registry pull artifacts.druid.gg/druid-team/scroll-lgsm:${TAG}server" 

docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v build-folder:/app/resources --entrypoint /app/resources/druid-install-command.sh -w /app/resources/deployment artifacts.druid.gg/druid-team/druidd-lgsm:$TAG registry pull artifacts.druid.gg/druid-team/scroll-lgsm:${TAG}server

echo "Running docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v build-folder:/app/resources --entrypoint /app/resources/druid-install-command.sh -w /app/resources/deployment artifacts.druid.gg/druid-team/druidd-lgsm:$TAG run install"

docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v build-folder:/app/resources --entrypoint /app/resources/druid-install-command.sh -w /app/resources/deployment artifacts.druid.gg/druid-team/druidd-lgsm:$TAG run install

echo "Prebuild done"

echo "Running docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v build-folder:/app/resources --entrypoint /app/resources/druid-install-command.sh -w /app/resources/deployment artifacts.druid.gg/druid-team/druidd-lgsm:$TAG backup $PRESIGNED_URL"
docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v build-folder:/app/resources --entrypoint /app/resources/druid-install-command.sh -w /app/resources/deployment artifacts.druid.gg/druid-team/druidd-lgsm:$TAG backup $PRESIGNED_URL
echo "Prebuild uploaded"


rm -rf tmp-prebuild
