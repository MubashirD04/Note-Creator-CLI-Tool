.PHONY: build install uninstall

build:
	go build -o notes-cli

install: build
	sudo cp notes-cli /usr/local/bin/notes-cli

uninstall:
	sudo rm -f /usr/local/bin/notes-cli
	rm -f /usr/local/bin/.env
