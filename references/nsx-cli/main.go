package main

import (
	_ "github.com/joho/godotenv/autoload"

	"github.com/NSXBet/nsx-cli/cmd"
)

// Version information is injected at build time
var version = "dev"

func main() {
	cmd.Execute(version)
}
