.PHONY: build install uninstall

build:
	go build -o notes-cli

install: build
	cp notes-cli /usr/local/bin/notes-cli

uninstall:
	rm -f /usr/local/bin/notes-cli
