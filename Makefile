.PHONY: install build run dbuild drun dbash containered

PROJECT = badbitchreads

install:
	go mod download

build:
	go build -a -race -v -o $(PROJECT) .

publish:
	GOOS=linux GOARCH=amd64 go build -o idoread.linux .
	scp -i ~/.ssh/idoread.com/rsync_id_ed25519 idoread.linux idoread@idoread.com:~/idoread.linux
