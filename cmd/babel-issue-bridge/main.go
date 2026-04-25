package main

import (
	"os"

	"babel-runtime/internal/ops/issuebridge"
)

func main() {
	os.Exit(issuebridge.Main(os.Args[1:], os.Stdout, os.Stderr))
}
