#!/bin/bash

if ! command -v nsx-gen &>/dev/null; then
    echo "nsx-gen could not be found! Installing..."

    go install github.com/NSXBet/nsx-gen@latest
fi

if ! command -v bufx &>/dev/null; then
    echo "bufx could not be found! Installing..."

    go install github.com/NSXBet/bufx@latest
fi

if ! command -v buf &>/dev/null; then
    echo "buf could not be found! Installing..."

    go install github.com/bufbuild/buf/cmd/buf@latest
fi

if ! command -v protoc-gen-go-cache-manager &>/dev/null; then
    echo "protoc-gen-go-cache-manager could not be found! Installing..."

    go install github.com/NSXBet/protoc-gen-go-cache-manager@latest
fi

if ! command -v mockery &>/dev/null; then
    echo "mockery could not be found! Installing..."

    go install github.com/vektra/mockery/v3@latest
fi


echo "✅ Setup gen complete."
