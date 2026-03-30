package setup

import (
	"fmt"

	cfg "github.com/edwin-Marrima/Tardigrade-runtime/pkg/config"
	pkgsetup "github.com/edwin-Marrima/Tardigrade-runtime/pkg/setup"
	"github.com/edwin-Marrima/Tardigrade-runtime/pkg/setup/rootfs"
	log "github.com/sirupsen/logrus"
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
			configPath, _ := cmd.Root().PersistentFlags().GetString("config")
			config, err := cfg.NewConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			reinstall, _ := cmd.Flags().GetBool("reinstall")

			segments := []pkgsetup.Segment{
				&pkgsetup.Runner{VmlinuxPath: config.Runtime.LinuxKernelPath, FirecrackerPath: config.Runtime.FirecrackerPath},
				&pkgsetup.CniSegment{Config: config},
				&pkgsetup.InitramfsSegment{Config: config},
				&rootfs.RootfsSegment{
					Image:          config.Filesystem.Rootfs.DockerImage,
					OutputFilePath: config.Filesystem.Rootfs.Path,
				},
				&pkgsetup.SystemdSegment{ConfigPath: configPath},
			}

			for _, seg := range segments {
				if reinstall {
					log.Infof("deprovisioning %T", seg)
					if err := seg.DeProvision(); err != nil {
						return fmt.Errorf("deprovision %T: %w", seg, err)
					}
				}
				log.Infof("provisioning %T", seg)
				if err := seg.Provision(); err != nil {
					return fmt.Errorf("provision %T: %w", seg, err)
				}
			}
			return nil
		},
	}

	cmd.Flags().Bool("reinstall", false, "Deprovision each segment before provisioning it")
	return cmd
}

func NewDestroyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "destroy",
		Short: "Removes all Tardigrade runtime components installed by the setup command",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Root().PersistentFlags().GetString("config")
			config, err := cfg.NewConfig(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			segments := []pkgsetup.Segment{
				&pkgsetup.SystemdSegment{ConfigPath: configPath},
				&rootfs.RootfsSegment{
					Image:          config.Filesystem.Rootfs.DockerImage,
					OutputFilePath: config.Filesystem.Rootfs.Path,
				},
				&pkgsetup.InitramfsSegment{Config: config},
				&pkgsetup.CniSegment{Config: config},
				&pkgsetup.Runner{VmlinuxPath: config.Runtime.LinuxKernelPath, FirecrackerPath: config.Runtime.FirecrackerPath},
			}

			for _, seg := range segments {
				log.Infof("deprovisioning %T", seg)
				if err := seg.DeProvision(); err != nil {
					return fmt.Errorf("deprovision %T: %w", seg, err)
				}
			}
			return nil
		},
	}
}
