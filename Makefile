.PHONY: build install uninstall

build:
	go build -o notes-cli

install: build
	sudo cp notes-cli /usr/local/bin/notes-cli
	if [ -f .env ]; then cp .env $(HOME)/.notes-cli.env; fi
	if [ -f notes.json ]; then cp notes.json $(HOME)/.notes-cli.json; fi

uninstall:
	sudo rm -f /usr/local/bin/notes-cli
	rm -f $(HOME)/.notes-cli.env
	rm -f $(HOME)/.notes-cli.json
