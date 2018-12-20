#!/bin/sh
if [ "$1" == "" ]; then
	echo ERR: version tag required as first param
else
	([ -d "$HOME/bw2/$1/data" ] || mkdir -p "$HOME/bw2/$1/data") && \
	GOOS=darwin GOARCH=amd64 go build -o "$HOME/bw2/$1/bin/darwin.amd64/bw2"  && \
	GOOS=linux GOARCH=amd64 go build -o "$HOME/bw2/$1/bin/linux.amd64/bw2"  && \
  rm -f "$HOME/bw2/$1/data/" *.jld && \
	cp conf.jlf "$HOME/bw2/$1/data" && \
  cp proj.conf.def.jlf "$HOME/bw2/$1/data" && \
	true
fi
