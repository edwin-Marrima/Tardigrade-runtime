package rootfs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"gvisor.dev/gvisor/pkg/cleanup"
)

const (
	rootfsImageArtifact = "rootfs.tar"
	containerName       = "tardigrade-rootfs-extract"
)

// run is a package-level variable so tests can replace it with a mock.
var run = func(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type RootfsSegment struct {
	Image          string
	OutputFilePath string
}

func (s *RootfsSegment) Provision() error {
	return SetupRootfs(s.Image, s.OutputFilePath)
}

func (s *RootfsSegment) DeProvision() error {
	if err := run("docker", "rm", containerName); err != nil {
		log.WithError(err).Warnf("failed to remove container %s", containerName)
	}
	if err := os.Remove(s.OutputFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove rootfs image: %w", err)
	}
	return nil
}

// SetupRootfs pulls image from the registry, exports its filesystem into a
// 2 GiB ext4 image written to outputFilePath.
func SetupRootfs(image, outputFilePath string) error {
	lg := log.WithFields(log.Fields{"image": image, "outputFilePath": outputFilePath})

	lg.Info("setting up rootfs")

	lg.Debug("creating container")
	if err := run("docker", "create", "--name", containerName, image); err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}
	cleaner := cleanup.Make(func() {
		_ = run("docker", "rm", "-f", containerName)
	})
	defer cleaner.Clean()

	tarPath := filepath.Join(os.TempDir(), rootfsImageArtifact)
	lg.WithField("output", tarPath).Debug("exporting container")
	if err := run("docker", "export", containerName, "-o", tarPath); err != nil {
		return fmt.Errorf("docker export: %w", err)
	}
	cleaner.Add(func() {
		_ = os.Remove(tarPath)
	})

	if err := run("fallocate", "-l", "2G", outputFilePath); err != nil {
		return fmt.Errorf("fallocate: %w", err)
	}
	cleaner.Add(func() {
		_ = os.Remove(outputFilePath)
	})

	if err := run("mkfs.ext4", outputFilePath); err != nil {
		return fmt.Errorf("mkfs.ext4: %w", err)
	}

	mountDir, err := os.MkdirTemp("", "tardigrade-rootfs-*")
	if err != nil {
		return fmt.Errorf("failed to create mount dir: %w", err)
	}
	cleaner.Add(func() {
		_ = os.RemoveAll(mountDir)
	})

	if err := run("mount", "-o", "loop", outputFilePath, mountDir); err != nil {
		return fmt.Errorf("mount: %w", err)
	}
	cleaner.Add(func() {
		_ = run("umount", mountDir)
	})

	if err := run("tar", "-xvf", tarPath, "-C", mountDir); err != nil {
		return fmt.Errorf("tar extract: %w", err)
	}
	cleaner.Release()
	return nil
}
