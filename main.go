package main

import (
	"fmt"
	"os"

	"github.com/taka-sho/teraflow/cmd"
)

var version = "v0.5.15"

func main() {
	if err := cmd.Execute(version); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
