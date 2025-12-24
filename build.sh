#!/bin/sh

cd frontend
npm i
npm run build

cd ..
echo "Backend"

mkdir -p web/html
rm -fr web/html/*
cp -R frontend/dist/* web/html/

go build -ldflags "-w -s" -tags "with_quic,with_utls,with_gvisor,with_wireguard" -o s-ui main.go
