package config

import (
	"errors"

	"github.com/spf13/viper"
)

type Config struct {
	Rootfs             string `mapstructure:"rootfs"`
	RootfsImage        string `mapstructure:"rootfs-image"`
	Initramfs          string `mapstructure:"initramfs"`
	Kernel             string `mapstructure:"kernel"`
	Port               int    `mapstructure:"port"`
	FireCrackerBinPath string `mapstructure:"firecracker-bin-path"`
	StatePath          string `mapstructure:"state-path"`
	CNINetworkName     string `mapstructure:"cni-network-name"`
	VMCidr             string `mapstructure:"vm-cidr"`
}

func NewConfig() (*Config, error) {
	viper.SetDefault("rootfs", "/opt/tardigrade/runtime/fs/rootfs")
	viper.SetDefault("rootfs-image", "")
	viper.SetDefault("initramfs", "/opt/tardigrade/runtime/fs/initramfs.cpio")
	viper.SetDefault("kernel", "/opt/tardigrade/runtime/bin/vmlinux")
	viper.SetDefault("port", 8080)
	viper.SetDefault("firecracker-bin-path", "/usr/local/bin/firecracker")
	viper.SetDefault("state-path", "/var/tardigrade/runtime/state")
	viper.SetDefault("cni-network-name", "tardigrade")
	viper.SetDefault("vm-cidr", "172.16.0.1/24")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/tardigrade/")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
