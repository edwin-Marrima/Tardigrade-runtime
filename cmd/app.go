package main

import (
	"github.com/edwin-Marrima/Tardigrade-runtime/cmd/app"
	"github.com/edwin-Marrima/Tardigrade-runtime/cmd/setup"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tardigrade-runtime",
		Aliases: []string{"tardigrade", "runtime", "tr"},
		Short:   "Tardigrade microVM runtime",
	}

	registerSharedFlags(cmd)

	cmd.AddCommand(app.NewServeCmd())
	cmd.AddCommand(app.NewInitramSetupCmd())
	cmd.AddCommand(setup.NewSetupCmd())

	return cmd
}

// registerSharedFlags adds flags that are shared across subcommands (serve, setup)
// as persistent flags on the root command so they are available to all children.
func registerSharedFlags(root *cobra.Command) {
	pf := root.PersistentFlags()

	pf.String("rootfs", "", "Path to the root filesystem image")
	pf.String("state-path", "/var/lib/tardigrade-runtime/", "Path to the virtual machine state")
	pf.String("firecracker-bin-path", "", "Path to firecracker binary")
	pf.String("initramfs", "", "Path to the initramfs image")
	pf.String("kernel", "", "Path to the kernel image")
	pf.Int("port", 8080, "Port to listen on")
	pf.String("vm-cidr", "", "CIDR range for micro vm in cluster")
	pf.String("cni-network-name", "", "CNI network name used when invoking CNI")

	viper.BindPFlag("rootfs", pf.Lookup("rootfs"))
	viper.BindPFlag("rootfs-image", pf.Lookup("rootfs-image"))
	viper.BindPFlag("state-path", pf.Lookup("state-path"))
	viper.BindPFlag("firecracker-bin-path", pf.Lookup("firecracker-bin-path"))
	viper.BindPFlag("initramfs", pf.Lookup("initramfs"))
	viper.BindPFlag("kernel", pf.Lookup("kernel"))
	viper.BindPFlag("port", pf.Lookup("port"))
	viper.BindPFlag("vm-cidr", pf.Lookup("vm-cidr"))
	viper.BindPFlag("cni-network-name", pf.Lookup("cni-network-name"))
}

func main() {
	if err := rootCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}
