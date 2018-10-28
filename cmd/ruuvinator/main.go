package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var version = "dev" // replaced dynamically at build time

func main() {
	app := &cobra.Command{
		Use:     os.Args[0],
		Short:   "Ruuvinator",
		Version: version,
	}

	app.AddCommand(btListenerEntry())
	app.AddCommand(metricsServerEntry())

	if err := app.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
