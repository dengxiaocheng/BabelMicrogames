package main

import (
	"os"

	"babel-runtime/internal/ops/collabmcp"
)

func main() {
	os.Exit(collabmcp.Main(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
