include .env
export

build-all:
	sh build-all-scrolls.sh

build-changed:
	sh build-changed-scrolls.sh

fetch-latest:
	sh fetch-latest.sh