#!/bin/sh
mkdir -p $HOME/bw2/$1/bin/darwin.amd64 && \
GOOS=darwin GOARCH=amd64 go build -o $HOME/bw2/$1/bin/darwin.amd64/bw2 bw2.go && \
mkdir -p $HOME/bw2/bin/$1/linux.amd64 && \
GOOS=linux GOARCH=amd64 go build -o $HOME/bw2/$1/bin/linux.amd64/bw2 bw2.go
