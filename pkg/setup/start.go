package setup

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"text/template"

	"github.com/edwin-Marrima/Tardigrade-runtime/pkg/config"
	log "github.com/sirupsen/logrus"
)

const binaryDest = "/usr/local/bin/tardigrade-runtime"

const serviceTemplate = `[Unit]
Description=Tardigrade microVM runtime
After=network.target

[Service]
Type=simple
ExecStart={{.BinaryPath}} serve \
  --rootfs {{.Rootfs}} \
  --initramfs {{.Initramfs}} \
  --kernel {{.Kernel}} \
  --port {{.Port}} \
  --firecracker-bin-path {{.FireCrackerBinPath}} \
  --state-path {{.StatePath}} \
  --cni-network-name {{.CNINetworkName}} \
  --vm-cidr {{.VMCidr}}
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
`

const serviceUnitPath = "/etc/systemd/system/tardigrade-runtime.service"

type serviceConfig struct {
	BinaryPath         string
	Rootfs             string
	Initramfs          string
	Kernel             string
	Port               int
	FireCrackerBinPath string
	StatePath          string
	CNINetworkName     string
	VMCidr             string
}

func Start(cfg *config.Config) error {
	if err := copyBinary(); err != nil {
		return fmt.Errorf("failed to install binary: %w", err)
	}

	if err := writeServiceUnit(cfg); err != nil {
		return fmt.Errorf("failed to write systemd unit: %w", err)
	}

	for _, args := range [][]string{
		{"daemon-reload"},
		{"enable", "tardigrade-runtime.service"},
		{"start", "tardigrade-runtime.service"},
	} {
		if err := runSystemctl(args...); err != nil {
			return err
		}
	}

	log.Info("tardigrade-runtime service started successfully")
	return nil
}

func copyBinary() error {
	src, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not resolve current executable: %w", err)
	}

	log.WithFields(log.Fields{"src": src, "dst": binaryDest}).Info("installing binary")

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(binaryDest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Sync()
}

func writeServiceUnit(cfg *config.Config) error {
	tmpl, err := template.New("service").Parse(serviceTemplate)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(serviceUnitPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data := serviceConfig{
		BinaryPath:         binaryDest,
		Rootfs:             cfg.Rootfs,
		Initramfs:          cfg.Initramfs,
		Kernel:             cfg.Kernel,
		Port:               cfg.Port,
		FireCrackerBinPath: cfg.FireCrackerBinPath,
		StatePath:          cfg.StatePath,
		CNINetworkName:     cfg.CNINetworkName,
		VMCidr:             cfg.VMCidr,
	}

	log.WithField("path", serviceUnitPath).Info("writing systemd service unit")
	return tmpl.Execute(f, data)
}

func runSystemctl(args ...string) error {
	cmd := exec.Command("systemctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.WithField("args", args).Info("running systemctl")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("systemctl %v: %w", args, err)
	}
	return nil
}
