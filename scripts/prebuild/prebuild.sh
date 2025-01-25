#!/bin/bash
set -e

TAG=$0

PRESIGN_OBJECT_KEY=lgsm/${TAG}-snapshot-latest.tar.gz

PRESIGNED_URL=$(cd scripts/presign/ && go run main.go)


rm -rf tmp-prebuild
mkdir -p tmp-prebuild/build

wget -O tmp-prebuild/druid-install-command.sh https://github.com/highcard-dev/druid-cli/releases/latest/download/druid-install-command.sh
chmod +x tmp-prebuild/druid-install-command.sh

docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v ./tmp-prebuild:/app/prebuild --entrypoint /app/prebuild/druid-install-command.sh -w /app/prebuild artifacts.druid.gg/druid-team/lgsm:$TAG registry pull artifacts.druid.gg/druid-team/lgsm:${TAG}server

docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v ./tmp-prebuild:/app/prebuild --entrypoint /app/prebuild/druid-install-command.sh -w /app/prebuild artifacts.druid.gg/druid-team/lgsm:$TAG run install
echo "Prebuild done"

docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v ./tmp-prebuild:/app/prebuild --entrypoint /app/prebuild/druid-install-command.sh -w /app/prebuild artifacts.druid.gg/druid-team/lgsm:$TAG backup $PRESIGNED_URL
echo "Prebuild uploaded"


rm -rf tmp-prebuild