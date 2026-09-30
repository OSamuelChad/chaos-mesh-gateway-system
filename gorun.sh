#!/bin/sh
# Executa comandos Go dentro do container golang:1.22-alpine (não exige Go instalado no host).
# Uso: ./gorun.sh <pasta-do-servico> "<comando>"
set -e
SERVICE="$1"; shift
docker run --rm \
  -v "$(cd "$(dirname "$0")" && pwd)":/src \
  -v chaos-mesh-gomod:/go/pkg/mod \
  -v chaos-mesh-gobuild:/root/.cache/go-build \
  -v chaos-mesh-gobin:/go/bin \
  -e GOTOOLCHAIN=local -e GOFLAGS=-buildvcs=false \
  -w "/src/$SERVICE" golang:1.22-alpine sh -c "$*"
