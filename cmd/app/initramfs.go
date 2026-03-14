package app

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	initramfsWorkDir = "/tmp/initramfs"
	initrdOutput     = "initrd.cpio"
	initrdTmp        = "/tmp/initrd.cpio"
)

func NewInitramSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "initram-setup",
		Short: "Build an initramfs cpio image",
		RunE: func(cmd *cobra.Command, args []string) error {
			shellBinPath, err := cmd.Flags().GetString("shell-bin-path")
			if err != nil {
				return err
			}
			if shellBinPath == "" {
				return fmt.Errorf("--shell-bin-path is required")
			}
			return runInitramSetup(shellBinPath)
		},
	}

	cmd.Flags().String("shell-bin-path", "", "Path to the busybox binary")

	return cmd
}

func runInitramSetup(shellBinPath string) error {
	dirs := []string{"bin", "dev", "etc", "home", "mnt", "proc", "sys", "usr"}
	for _, d := range dirs {
		path := filepath.Join(initramfsWorkDir, d)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create dir %s: %w", path, err)
		}
	}

	if err := copyFile(shellBinPath, filepath.Join(initramfsWorkDir, "bin", "busybox"), 0755); err != nil {
		return fmt.Errorf("failed to copy busybox: %w", err)
	}

	if err := copyFile("initramfs-script.sh", filepath.Join(initramfsWorkDir, "init"), 0755); err != nil {
		return fmt.Errorf("failed to copy initramfs-script.sh: %w", err)
	}

	log.Infof("initramfs work dir contents: %s", initramfsWorkDir)
	entries, _ := os.ReadDir(initramfsWorkDir)
	for _, e := range entries {
		log.Info(e.Name())
	}

	if err := buildCpio(); err != nil {
		return fmt.Errorf("failed to build cpio: %w", err)
	}

	if err := os.Rename(initrdTmp, initrdOutput); err != nil {
		return fmt.Errorf("failed to move cpio to output: %w", err)
	}

	if err := os.Chmod(initrdOutput, 0755); err != nil {
		return fmt.Errorf("failed to chmod output: %w", err)
	}

	if err := os.RemoveAll(initramfsWorkDir); err != nil {
		return fmt.Errorf("failed to clean up work dir: %w", err)
	}

	log.Infof("initramfs image written to %s", initrdOutput)
	return nil
}

func buildCpio() error {
	outFile, err := os.Create(initrdTmp)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	find := exec.Command("find", ".", "-print0")
	find.Dir = initramfsWorkDir

	cpio := exec.Command("cpio", "--null", "--create", "--verbose", "--format=newc")
	cpio.Dir = initramfsWorkDir
	cpio.Stdout = outFile

	cpio.Stdin, err = find.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to pipe find to cpio: %w", err)
	}

	if err := cpio.Start(); err != nil {
		return fmt.Errorf("failed to start cpio: %w", err)
	}
	if err := find.Run(); err != nil {
		return fmt.Errorf("find failed: %w", err)
	}
	if err := cpio.Wait(); err != nil {
		return fmt.Errorf("cpio failed: %w", err)
	}

	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
