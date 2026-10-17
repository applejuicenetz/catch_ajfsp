#!/bin/sh
# Build catch_ajfsp for Windows (amd64 and arm64) inside Docker. Output: dist/
set -eu
cd "$(dirname "$0")"
docker run --rm -v "$PWD":/src -w /src golang:1.27 sh -c '
  set -eu
  test -z "$(gofmt -l .)"
  go vet ./...
  go test ./internal/...
  GOOS=windows GOARCH=amd64 go vet ./...
  GOOS=windows GOARCH=arm64 go vet ./...
  for arch in amd64 arm64; do
    go run github.com/akavel/rsrc@v0.10.2 -arch $arch -ico cmd/catch_ajfsp/catch_ajfsp.ico \
      -manifest cmd/catch_ajfsp/catch_ajfsp.manifest -o cmd/catch_ajfsp/resource_windows_$arch.syso
    GOOS=windows GOARCH=$arch CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags "-s -w -H windowsgui" \
      -o dist/catch_ajfsp-windows-$arch.exe ./cmd/catch_ajfsp
  done
  chown -R '"$(id -u):$(id -g)"' dist cmd
'
