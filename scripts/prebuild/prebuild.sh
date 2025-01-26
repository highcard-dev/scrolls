#!/bin/bash
set -e

TAG=$1
echo "Tag: $TAG"

PRESIGN_OBJECT_KEY=lgsm/${TAG}-snapshot-latest.tar.gz

PRESIGNED_URL=$(cd scripts/presign/ && go run main.go)


rm -rf tmp-prebuild
mkdir -p tmp-prebuild/build
chmod 777 -R tmp-prebuild

wget -O tmp-prebuild/druid-install-command.sh https://github.com/highcard-dev/druid-cli/releases/latest/download/druid-install-command.sh
chmod +x tmp-prebuild/druid-install-command.sh

DOCKER_ENTRYPOINT=/app/prebuild/druid-install-command.sh

echo "Running docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v ./tmp-prebuild:/app/prebuild --entrypoint /app/prebuild/druid-install-command.sh -w /app/prebuild artifacts.druid.gg/druid-team/druidd-lgsm:$TAG registry pull artifacts.druid.gg/druid-team/scroll-lgsm:${TAG}server" 

docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v ./tmp-prebuild:/app/prebuild --entrypoint /app/prebuild/druid-install-command.sh -w /app/prebuild artifacts.druid.gg/druid-team/druidd-lgsm:$TAG registry pull artifacts.druid.gg/druid-team/scroll-lgsm:${TAG}server

echo "Running docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v ./tmp-prebuild:/app/prebuild --entrypoint /app/prebuild/druid-install-command.sh -w /app/prebuild artifacts.druid.gg/druid-team/druidd-lgsm:$TAG run install"

docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v ./tmp-prebuild:/app/prebuild --entrypoint /app/prebuild/druid-install-command.sh -w /app/prebuild artifacts.druid.gg/druid-team/druidd-lgsm:$TAG run install

echo "Prebuild done"

echo "Running docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v ./tmp-prebuild:/app/prebuild --entrypoint /app/prebuild/druid-install-command.sh -w /app/prebuild artifacts.druid.gg/druid-team/druidd-lgsm:$TAG backup $PRESIGNED_URL"
docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v ./tmp-prebuild:/app/prebuild --entrypoint /app/prebuild/druid-install-command.sh -w /app/prebuild artifacts.druid.gg/druid-team/druidd-lgsm:$TAG backup $PRESIGNED_URL
echo "Prebuild uploaded"


rm -rf tmp-prebuild

