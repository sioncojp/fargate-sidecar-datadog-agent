#!/bin/bash -
declare -r Name="fargate-sidecar-datadog-agent"

for GOOS in darwin linux; do
    GO111MODULE=on GOOS=$GOOS GOARCH=amd64 go build -o bin/fargate-sidecar-datadog-agent-$GOOS-amd64 *.go
done
