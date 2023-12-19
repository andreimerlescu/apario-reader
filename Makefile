.PHONY: install build run dbuild drun dbash containered

PROJECT = badbitchreads

install:
	go mod download

build:
	go build -a -race -v -o $(PROJECT) .


