#!/bin/sh
if [ "$1" == "" ]; then
		echo ERR: version tag required as first param
else
		GOOS=darwin GOARCH=amd64 go build -o "$HOME/bw2/$1/bin/darwin.amd64/bw2"  && \
		GOOS=linux GOARCH=amd64 go build -o "$HOME/bw2/$1/bin/linux.amd64/bw2"  && \
		cp project.defs "$HOME/bw2/$1"
fi
