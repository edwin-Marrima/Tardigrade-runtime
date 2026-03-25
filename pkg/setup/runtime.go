package setup

import (
	"fmt"
	"os"

	bin "github.com/edwin-Marrima/Tardigrade-runtime/.bin"
	"github.com/edwin-Marrima/Tardigrade-runtime/pkg/utils"
	log "github.com/sirupsen/logrus"
)

type Runner struct {
	VmlinuxPath     string
	FirecrackerPath string
}

func (s *Runner) Provision() error {
	log.WithField("path", s.VmlinuxPath).Info("setting up vmlinux")
	if err := utils.WriteFile(bin.Vmlinux, s.VmlinuxPath); err != nil {
		return fmt.Errorf("failed to install vmlinux at %s: %w", s.VmlinuxPath, err)
	}

	log.WithField("path", s.FirecrackerPath).Info("setting up firecracker")
	if err := utils.WriteFile(bin.Firecracker, s.FirecrackerPath); err != nil {
		return fmt.Errorf("failed to install firecracker at %s: %w", s.FirecrackerPath, err)
	}
	if err := os.Chmod(s.FirecrackerPath, 0755); err != nil {
		return fmt.Errorf("failed to chmod firecracker at %s: %w", s.FirecrackerPath, err)
	}
	return nil
}

func (s *Runner) DeProvision() error {
	log.WithField("path", s.VmlinuxPath).Info("removing vmlinux")
	if err := os.Remove(s.VmlinuxPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove vmlinux at %s: %w", s.VmlinuxPath, err)
	}

	log.WithField("path", s.FirecrackerPath).Info("removing firecracker")
	if err := os.Remove(s.FirecrackerPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove firecracker at %s: %w", s.FirecrackerPath, err)
	}
	return nil
}
