#!/bin/bash

if ! command -v golangci-lint &>/dev/null; then
    echo "golangci-lint could not be found! Installing..."

    go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
fi

if ! command -v yamllint &>/dev/null; then
    echo "yamllint could not be found! Installing..."

    pip install --user yamllint
fi

echo "✅ Setup linting complete."
