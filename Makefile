.PHONY: install build publish

PROJECT = badbitchreads

install:
	go mod download

build:
	mkdir -p bin
	GOOS=linux GOARCH=amd64 go build -a -v -o bin/idoread.linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -a -v -o bin/idoread.macos-amd64 .
	GOOS=windows GOARCH=amd64 go build -a -v -o bin/idoread.windows-amd64.exe .

publish: build
	scp -i ~/.ssh/idoread.com/rsync_id_ed25519 bin/idoread.linux-amd64 idoread@idoread.com:~/idoread.linux-amd64.v1.0.2
