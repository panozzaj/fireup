#!/bin/bash
cd /Users/anthony/Documents/dev/roost-dev
unset GOPATH
go test ./... "$@"
