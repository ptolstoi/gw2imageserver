#!/bin/sh

go build -v -o bin/${PWD##*/} \
  -mod vendor \
  -ldflags "-X main._buildTime=`date -u +%Y-%m-%d_%H-%M-%S` 
    -X main._version=`cat version`.`git rev-parse --short HEAD` 
    -X main._showStacktrace=set" \
  ./cmd/${PWD##*/}