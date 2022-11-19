
build:
	go build  -o ./.bin/scrolls-registry ./scrolls-registry/main.go

build-all: build
	 ./.bin/scrolls-registry build -e ./.env -d ./scrolls

build-only-changed: build
	 ./.bin/scrolls-registry build -e ./.env -d ./scrolls -c

print: build
	 ./.bin/scrolls-registry print -e ./.env