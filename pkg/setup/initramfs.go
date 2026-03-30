package setup

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	bin "github.com/edwin-Marrima/Tardigrade-runtime/.bin"
	"github.com/edwin-Marrima/Tardigrade-runtime/pkg/config"
	"github.com/edwin-Marrima/Tardigrade-runtime/pkg/utils"
	log "github.com/sirupsen/logrus"
)

const (
	defaultInitramfsWorkDir = "/tmp/initramfs"
)

//go:embed initramfs.sh
var initRamFS []byte

//go:embed initramfs-builder.sh
var initRamFSBuilder []byte

type InitramfsSegment struct {
	Config *config.Config
}

func (s *InitramfsSegment) Provision() error {
	return runInitRamFSBuilder(s.Config.Filesystem.InitramPath, defaultInitramfsWorkDir)
}

func (s *InitramfsSegment) DeProvision() error {
	if err := os.Remove(s.Config.Filesystem.InitramPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove initramfs: %w", err)
	}
	return nil
}

func CreateInitRamFS(cfg *config.Config) error {
	return (&InitramfsSegment{Config: cfg}).Provision()
}

func runInitRamFSBuilder(outFile, workDir string) error {
	lg := log.WithFields(log.Fields{"work.dir": workDir, "output.file": outFile})
	lg.Info("Running initramfs builder")

	tmpDir, err := os.MkdirTemp("", "initramfs-build-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	lg.WithField("temp.dir", tmpDir).Debug("copying busybox binary to temp dir")
	if err := utils.WriteFile(bin.Busybox, filepath.Join(tmpDir, "busybox")); err != nil {
		return fmt.Errorf("failed to write busybox: %w", err)
	}

	lg.Debug("copying initramfs binary")
	if err := utils.WriteFile(initRamFS, filepath.Join(tmpDir, "initramfs.sh")); err != nil {
		return fmt.Errorf("failed to write initramfs.sh: %w", err)
	}

	builderPath := filepath.Join(tmpDir, "initramfs-builder.sh")
	if err := utils.WriteFile(initRamFSBuilder, builderPath); err != nil {
		return fmt.Errorf("failed to write initramfs-builder.sh: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outFile), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	cmd := exec.Command("/bin/bash", builderPath)
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"INITRAMFS_WORK_DIR="+workDir,
		"OUT_FILE="+outFile,
	)
	lg.Debug("executing initramfs build script")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run initramfs builder: %w", err)
	}

	lg.Info("initramfs successfully built")
	return nil
}
