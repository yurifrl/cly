#!/bin/bash

if ! command -v gosimports &>/dev/null; then
    echo "gosimports could not be found! Installing..."

    go install github.com/rinchsan/gosimports/cmd/gosimports@latest
fi

if ! command -v golines &>/dev/null; then
    echo "golines could not be found! Installing..."

    go install github.com/segmentio/golines@latest
fi

if ! command -v gofumpt &>/dev/null; then
    echo "gofumpt could not be found! Installing..."

    go install mvdan.cc/gofumpt@latest
fi

if ! command -v yamlfmt &>/dev/null; then
    echo "yamlfmt could not be found! Installing..."

    go install github.com/google/yamlfmt/cmd/yamlfmt@latest
fi

if ! command -v buf &>/dev/null; then
    echo "buf could not be found! Installing..."

    go install github.com/bufbuild/buf/cmd/buf@latest
fi

if ! command -v bufx &>/dev/null; then
    echo "bufx could not be found! Installing..."

    go install github.com/NSXBet/bufx@latest
fi

echo "✅ Setup formatting complete."
