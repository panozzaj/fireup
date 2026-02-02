#!/bin/bash
cd /Users/anthony/Documents/dev/fireup
unset GOPATH
go test ./... "$@"
