package setup

import (
	"fmt"

	cfg "github.com/edwin-Marrima/Tardigrade-runtime/pkg/config"
	pkgsetup "github.com/edwin-Marrima/Tardigrade-runtime/pkg/setup"
	"github.com/edwin-Marrima/Tardigrade-runtime/pkg/setup/rootfs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// it responsible for setting up the requirements to run tardigrade-runtime.
// - CNI - configuration and binaries
// - initramfs
// - rootfs setup and /init

func NewSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "setup",
		Short: "Initializes the core dependencies for the Tardigrade runtime. This process provisions " +
			"the container network interface (CNI) plugins, the initial ramdisk (initramfs), and the rootfs",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := cfg.NewConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			if err := pkgsetup.Cni(config); err != nil {
				return fmt.Errorf("failed to setup CNI: %w", err)
			}
			if err := rootfs.SetupRootfs(config.RootfsImage, config.Rootfs); err != nil {
				return fmt.Errorf("failed to setup rootfs: %w", err)
			}
			if err := pkgsetup.CreateInitRamFS(config); err != nil {
				return fmt.Errorf("failed to setup InitRamfs: %w", err)
			}
			if err := pkgsetup.Start(config); err != nil {
				return fmt.Errorf("failed to setup systemd service: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().String("rootfs", "", "Path to the root filesystem image")
	cmd.Flags().String("rootfs-image", "", "Path to the rootfs base image")
	cmd.Flags().String("state-path", "/var/lib/tardigrade-runtime/", "Path to the virtual machine state")
	cmd.Flags().String("firecracker-bin-path", "", "Path to firecracker binary")
	cmd.Flags().String("initramfs", "", "Path to the initramfs image")
	cmd.Flags().String("kernel", "", "Path to the kernel image")
	cmd.Flags().Int("port", 8080, "Port to listen on")
	cmd.Flags().String("vm-cidr", "", "CIDR range for micro vm in cluster")
	cmd.Flags().String("cni-network-name", "", "CNI network name used when invoking CNI")

	viper.BindPFlag("rootfs", cmd.Flags().Lookup("rootfs"))
	viper.BindPFlag("rootfs-image", cmd.Flags().Lookup("rootfs-image"))
	viper.BindPFlag("state-path", cmd.Flags().Lookup("state-path"))
	viper.BindPFlag("firecracker-bin-path", cmd.Flags().Lookup("firecracker-bin-path"))
	viper.BindPFlag("initramfs", cmd.Flags().Lookup("initramfs"))
	viper.BindPFlag("kernel", cmd.Flags().Lookup("kernel"))
	viper.BindPFlag("port", cmd.Flags().Lookup("port"))
	viper.BindPFlag("cni-network-name", cmd.Flags().Lookup("cni-network-name"))
	viper.BindPFlag("vm-cidr", cmd.Flags().Lookup("vm-cidr"))

	return cmd
}
