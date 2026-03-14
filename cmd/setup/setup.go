package setup

import (
	"fmt"

	cfg "github.com/edwin-Marrima/Tardigrade-runtime/pkg/config"
	pkgsetup "github.com/edwin-Marrima/Tardigrade-runtime/pkg/setup"
	"github.com/spf13/cobra"
)

// it responsible for setting up the requirements to run tardigrade-runtime.
// - CNI - configuration and binaries
// - initramfs
// - rootfs setup and /init

func NewSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Set up tardigrade runtime requirements (CNI plugins, config, etc.)",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := cfg.NewConfig()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			if err := pkgsetup.Cni(config); err != nil {
				return fmt.Errorf("failed to setup CNI: %w", err)
			}
			return nil
		},
	}
}
