#!/bin/sh

set -e

echo "Building backend..."
go build -ldflags "-w -s" -tags "with_quic,with_utls,with_gvisor,with_wireguard" -o s-ui main.go

echo "Done: ./s-ui"
