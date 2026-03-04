package config

import (
	"errors"

	"github.com/spf13/viper"
)

type Config struct {
	Rootfs             string `mapstructure:"rootfs"`
	Initramfs          string `mapstructure:"initramfs"`
	Kernel             string `mapstructure:"kernel"`
	Port               int    `mapstructure:"port"`
	GuestVMPath        string `mapstructure:"guest-vm-path"`
	FireCrackerBinPath string `mapstructure:"firecracker-bin-path"`
	StatePath          string `mapstructure:"state-path"`
}

func NewConfig() (*Config, error) {
	viper.SetDefault("port", 8080)

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
