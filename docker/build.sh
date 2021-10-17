#!/bin/sh

set -eu

REPO=mikegleasonjr
NAME=dnswarmer
VERSION=0.0.1
PLATFORMS=linux/arm/v7,linux/arm64/v8,linux/amd64
IMAGE="${REPO}/${NAME}"

docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
docker buildx create --use --driver docker-container --name ${NAME} --node ${NAME}0
docker buildx build --platform $PLATFORMS -t "${IMAGE}:${VERSION}" -t "${IMAGE}:latest" --push .
