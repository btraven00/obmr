package main

import (
	"fmt"
	"os"

	"github.com/btraven00/obflow/cmd"
)

func main() {
	c := cmd.NewRunCmd()
	c.Use = "obrun"
	c.Short = "run an omnibenchmark (obflow run)"
	if err := c.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
