package main

import (
	"fmt"
	"os"

	"github.com/yurifrl/cly/modules/mcp"
)

var version = "dev"

func main() {
	cmd := mcp.NewRootCmd()
	cmd.Version = version
	cmd.SetVersionTemplate(fmt.Sprintf("mcp %s\n", version))

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
