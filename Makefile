
build-app:
	go build  -o ./.bin/scrolls-registry ./scrolls-registry/main.go

build-all: build-app
	 ./.bin/scrolls-registry build -e ./.env -d ./scrolls

build-only-changed: build-app
	 ./.bin/scrolls-registry build -e ./.env -d ./scrolls -c

push-all: build-app build-all
	 ./.bin/scrolls-registry push -e ./.env -d ./scrolls -t packages
	 ./.bin/scrolls-registry push -e ./.env -d ./scrolls -t registry-index
	 ./.bin/scrolls-registry push -e ./.env -d ./scrolls -t translations
	 ./.bin/scrolls-registry clean

push-only-changed: build-app build-only-changed
	 ./.bin/scrolls-registry push -e ./.env -d ./scrolls -t packages
	 ./.bin/scrolls-registry push -e ./.env -d ./scrolls -t registry-index
	 ./.bin/scrolls-registry push -e ./.env -d ./scrolls -t translations
	 ./.bin/scrolls-registry clean

print: build-app
	 ./.bin/scrolls-registry print -e ./.env

clean: build-app
	 ./.bin/scrolls-registry clean