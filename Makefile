
build-app:
	go install  -o ./.bin/scrolls-registry ./scrolls-registry/scroll-registry.go

install-app:
	go install  ./scrolls-registry/scroll-registry.go

build-all: install-app
	 scroll-registry build -e ./.env -d ./scrolls

build-only-changed: install-app
	 scroll-registry build -e ./.env -d ./scrolls -c

push-all: install-app build-all
	 scroll-registry push -e ./.env -d ./scrolls -t packages
	 scroll-registry push -e ./.env -d ./scrolls -t registry-index
	 scroll-registry push -e ./.env -d ./scrolls -t translations
	 scroll-registry clean

push-only-changed: install-app build-only-changed
	 scroll-registry push -e ./.env -d ./scrolls -t packages
	 scroll-registry push -e ./.env -d ./scrolls -t registry-index
	 scroll-registry push -e ./.env -d ./scrolls -t translations
	 scroll-registry clean

print: install-app
	 scroll-registry print -e ./.env

clean: install-app
	 scroll-registry clean