package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type RuntimeConfig struct {
	FirecrackerPath string `yaml:"firecracker_path"`
	StateFolder     string `yaml:"state_folder"`
	Port            int    `yaml:"port"`
	LinuxKernelPath string `yaml:"linux_kernel_path"`
}

type NetworkConfig struct {
	Cidr        string `yaml:"cidr"`
	NetworkName string `yaml:"network_name"`
}

type RootfsConfig struct {
	Path        string `yaml:"path"`
	DockerImage string `yaml:"docker_image"`
}

type FilesystemConfig struct {
	InitramPath string       `yaml:"initram_path"`
	Rootfs      RootfsConfig `yaml:"rootfs"`
}

type Config struct {
	Runtime    RuntimeConfig    `yaml:"runtime"`
	Network    NetworkConfig    `yaml:"network"`
	Filesystem FilesystemConfig `yaml:"filesystem"`
}

func defaults() *Config {
	return &Config{
		Runtime: RuntimeConfig{
			FirecrackerPath: "/usr/local/bin/firecracker",
			StateFolder:     "/var/tardigrade/runtime/state",
			Port:            8080,
			LinuxKernelPath: "/opt/tardigrade/runtime/bin/vmlinux",
		},
		Network: NetworkConfig{
			Cidr:        "172.16.0.0/24",
			NetworkName: "tardigrade",
		},
		Filesystem: FilesystemConfig{
			InitramPath: "/opt/tardigrade/runtime/fs/initramfs.cpio",
			Rootfs: RootfsConfig{
				Path:        "/opt/tardigrade/runtime/fs/rootfs.img",
				DockerImage: "tardigrade/rootfs:latest",
			},
		},
	}
}

func NewConfig(path string) (*Config, error) {
	cfg := defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	return cfg, nil
}
