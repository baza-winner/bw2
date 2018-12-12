#!/bin/sh
GOOS=darwin GOARCH=amd64 go build -o $HOME/bw2/bin/darwin.amd64/bw2 bw2.go && GOOS=linux GOARCH=amd64 go build -o $HOME/bw2/bin/linux.amd64/bw2 bw2.go