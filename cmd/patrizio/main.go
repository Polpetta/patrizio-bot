// Package main is the entrypoint for the Patrizio Delta Chat bot.
package main

import (
	"os"

	"github.com/polpetta/patrizio/internal/bot"
	"github.com/polpetta/patrizio/internal/config"
	"github.com/polpetta/patrizio/migrations"
)

func main() {
	config.Load()

	cli := bot.Setup(migrations.FS, ".")

	if err := cli.Start(); err != nil {
		cli.Logger.Error(err)
		os.Exit(1)
	}
}
