package setup

import (
	"fmt"

	cfg "github.com/edwin-Marrima/Tardigrade-runtime/pkg/config"
	pkgsetup "github.com/edwin-Marrima/Tardigrade-runtime/pkg/setup"
	"github.com/edwin-Marrima/Tardigrade-runtime/pkg/setup/rootfs"
	"github.com/spf13/cobra"
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
			if err := pkgsetup.CreateInitRamFS(config); err != nil {
				return fmt.Errorf("failed to setup InitRamfs: %w", err)
			}
			if err := rootfs.SetupRootfs(config.RootfsImage, config.Rootfs); err != nil {
				return fmt.Errorf("failed to setup rootfs: %w", err)
			}
			if err := pkgsetup.Start(config); err != nil {
				return fmt.Errorf("failed to setup systemd service: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().String("rootfs-image", "", "Image that contains the root filesystem content")
	return cmd
}
