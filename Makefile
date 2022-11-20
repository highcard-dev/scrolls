
build-app:
	go install  -o ./.bin/scrolls-registry ./scrolls-registry/scroll-registry.go

install-app:
	go install  ./scrolls-registry/scroll-registry.go

build-all: install-app
	 scrolls-registry build -e ./.env -d ./scrolls

build-only-changed: install-app
	 scrolls-registry build -e ./.env -d ./scrolls -c

push-all: install-app build-all
	 scrolls-registry push -e ./.env -d ./scrolls -t packages
	 scrolls-registry push -e ./.env -d ./scrolls -t registry-index
	 scrolls-registry push -e ./.env -d ./scrolls -t translations
	 scrolls-registry clean

push-only-changed: install-app build-only-changed
	 scrolls-registry push -e ./.env -d ./scrolls -t packages
	 scrolls-registry push -e ./.env -d ./scrolls -t registry-index
	 scrolls-registry push -e ./.env -d ./scrolls -t translations
	 scrolls-registry clean

print: install-app
	 scrolls-registry print -e ./.env

clean: install-app
	 scrolls-registry clean