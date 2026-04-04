#!/bin/bash
# MailVault build script for Wails v3
set -e

echo "==> Generating bindings..."
wails3 generate bindings

echo "==> Building frontend..."
cd frontend && npm run build -q && cd ..

echo "==> Building Go binary..."
mkdir -p bin
CGO_ENABLED=1 CGO_CFLAGS="-mmacosx-version-min=10.15" CGO_LDFLAGS="-mmacosx-version-min=10.15" \
  MACOSX_DEPLOYMENT_TARGET=10.15 \
  go build -tags production -trimpath -buildvcs=false -ldflags="-w -s" -o bin/mailvault .

echo "==> Done: bin/mailvault"
