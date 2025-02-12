#!/usr/bin/env bash

set -e

docker-compose up -d
go test -coverprofile=./c.out -v -race ./...
cp c.out coverage.txt
docker-compose down -v