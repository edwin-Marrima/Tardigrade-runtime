package setup

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"text/template"

	log "github.com/sirupsen/logrus"
)

const binaryDest = "/usr/local/bin/tardigrade-runtime"

const serviceTemplate = `[Unit]
Description=Tardigrade microVM runtime
After=network.target

[Service]
Type=simple
ExecStart={{.BinaryPath}} serve --config {{.ConfigPath}}
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
`

const serviceUnitPath = "/etc/systemd/system/tardigrade-runtime.service"

type serviceConfig struct {
	BinaryPath string
	ConfigPath string
}

type SystemdSegment struct {
	ConfigPath string
}

func (s *SystemdSegment) Provision() error {
	if err := copyBinary(); err != nil {
		return fmt.Errorf("failed to install binary: %w", err)
	}

	if err := writeServiceUnit(s.ConfigPath); err != nil {
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

func (s *SystemdSegment) DeProvision() error {
	for _, args := range [][]string{
		{"stop", "tardigrade-runtime.service"},
		{"disable", "tardigrade-runtime.service"},
	} {
		if err := runSystemctl(args...); err != nil {
			log.WithError(err).Warn("failed to stop tardigrade-runtime.service")
		}
	}

	if err := os.Remove(serviceUnitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove service unit: %w", err)
	}

	if err := os.Remove(binaryDest); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove binary: %w", err)
	}

	return runSystemctl("daemon-reload")
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

func writeServiceUnit(configPath string) error {
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
		BinaryPath: binaryDest,
		ConfigPath: configPath,
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
