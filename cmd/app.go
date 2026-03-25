package main

import (
	"github.com/edwin-Marrima/Tardigrade-runtime/cmd/app"
	"github.com/edwin-Marrima/Tardigrade-runtime/cmd/setup"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tardigrade-runtime",
		Aliases: []string{"tardigrade", "runtime", "tr"},
		Short:   "Tardigrade microVM runtime",
	}

	cmd.PersistentFlags().String("config", "/etc/tardigrade/configuration.yaml", "Path to the configuration file")

	cmd.AddCommand(app.NewServeCmd())
	cmd.AddCommand(app.NewInitramSetupCmd())
	cmd.AddCommand(setup.NewSetupCmd())
	cmd.AddCommand(setup.NewDestroyCmd())

	return cmd
}

func main() {
	if err := rootCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}
