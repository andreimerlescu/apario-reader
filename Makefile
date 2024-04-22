.PHONY: install build run dbuild drun dbash containered

PROJECT = badbitchreads

install:
	go mod download

build:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -a -v -o bin/idoread.linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -a -v -o bin/idoread.macos-amd64 .
	GOOS=windows GOARCH=amd64 go build -a -v -o bin/idoread.windows-amd64.exe .

publish:
	GOOS=linux GOARCH=amd64 go build -o idoread.linux .
	scp -i ~/.ssh/idoread.com/rsync_id_ed25519 idoread.linux idoread@idoread.com:~/idoread.linux
