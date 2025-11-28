#!/bin/bash
set -e

TAG=$1
echo "Tag: $TAG"

PRESIGN_OBJECT_KEY=lgsm/${TAG}-snapshot-latest.tar.gz

TMP_VOLUME_NAME=lgsm-prebuild-$(date +%s)

docker volume rm $TMP_VOLUME_NAME || true

docker run --rm -v $TMP_VOLUME_NAME:/app/resources bash sh -c 'wget -O /app/resources/druid-install-command.sh https://github.com/highcard-dev/druid-cli/releases/latest/download/druid-install-command.sh && mkdir /app/resources/deployment && chmod +x /app/resources/druid-install-command.sh && chown 1000:1000 -R /app/resources/'

DOCKER_ENTRYPOINT=/app/resources/druid-install-command.sh


echo "Pulling scroll"
docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v $TMP_VOLUME_NAME:/app/resources --entrypoint /app/resources/druid-install-command.sh -w /app/resources/deployment artifacts.druid.gg/druid-team/druid:latest-nix-steamcmd registry pull artifacts.druid.gg/druid-team/scroll-lgsm:${TAG}server

echo "Running scroll install script"
docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v $TMP_VOLUME_NAME:/app/resources --entrypoint /app/resources/druid-install-command.sh -w /app/resources/deployment artifacts.druid.gg/druid-team/druid:latest-nix-steamcmd run install

echo "Creating archive and uploading to s3"
docker run --entrypoint $DOCKER_ENTRYPOINT --rm -v $TMP_VOLUME_NAME:/app/resources --entrypoint /app/resources/druid-install-command.sh -w /app/resources/deployment artifacts.druid.gg/druid-team/druid:latest-nix-steamcmd backup $PRESIGN_OBJECT_KEY --s3-access-key $PRESIGN_ACCESS_KEY --s3-secret-key $PRESIGN_SECRET_KEY --s3-bucket $PRESIGN_BUCKET_NAME --s3-endpoint $PRESIGN_S3_ENDPOINT $BACKUP_ADDITIONAL_ARGS
echo "Prebuild uploaded"


docker volume rm $TMP_VOLUME_NAME || true
