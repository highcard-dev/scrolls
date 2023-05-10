
build-app:
	go install  -o ./.bin/druid registry ./druid/scroll-registry.go

install-app:
	go install  ./druid/scroll-registry.go

build-all: install-app
	 scroll-registry build -e ./.env -d ./scrolls

push-all: install-app build-all
	 scroll-registry push -e ./.env -d ./scrolls -t packages
	 scroll-registry push -e ./.env -d ./scrolls -t registry-index
	 scroll-registry push -e ./.env -d ./scrolls -t translations
	 scroll-registry clean

print: install-app
	 scroll-registry print -e ./.env

clean: install-app
	 scroll-registry clean