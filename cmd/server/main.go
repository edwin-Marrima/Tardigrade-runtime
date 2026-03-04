package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/edwin-Marrima/Tardigrade-runtime/pkg/config"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Server struct {
	config *config.Config
}

func NewServer(cfg *config.Config) *Server {
	return &Server{config: cfg}
}

func (s *Server) create() func(c *gin.Context) {
	return func(c *gin.Context) {}
}
func (s *Server) delete() func(c *gin.Context) {
	return func(c *gin.Context) {}
}

func cli(cfg *config.Config) *cobra.Command {
	var apiServer = &cobra.Command{
		Use:     "tardigrade-runtime",
		Aliases: []string{"tardigrade", "runtime", "tr"},
		Short:   "Runs the tardigrade runtime API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			cfg, err = config.NewConfig()
			if err != nil {
				log.WithError(err).Error("failed to load config")
				return fmt.Errorf("failed to load config: %w", err)
			}
			return nil
		},
	}
	apiServer.Flags().String("rootfs", "", "Path to the root filesystem image")
	apiServer.Flags().String("state-path", "/var/lib/tardigrade-runtime/", "Path to the virtual machine state")
	apiServer.Flags().String("firecracker-bin-path", "", "Path to firecracker binary")
	apiServer.Flags().String("initramfs", "", "Path to the initramfs image")
	apiServer.Flags().String("kernel", "", "Path to the kernel image")
	apiServer.Flags().Int("port", 8080, "Port to listen on")
	apiServer.Flags().String("guest-vm-path", "", "Path to the guest VM directory")

	viper.BindPFlag("rootfs", apiServer.Flags().Lookup("rootfs"))
	viper.BindPFlag("firecracker-bin-path", apiServer.Flags().Lookup("firecracker-bin-path"))
	viper.BindPFlag("initramfs", apiServer.Flags().Lookup("initramfs"))
	viper.BindPFlag("kernel", apiServer.Flags().Lookup("kernel"))
	viper.BindPFlag("port", apiServer.Flags().Lookup("port"))
	viper.BindPFlag("guest-vm-path", apiServer.Flags().Lookup("guest-vm-path"))

	return apiServer
}
func main() {
	var cfg *config.Config
	cmd := cli(cfg)
	err := cmd.Execute()
	if err != nil {
		log.Fatal(err)
	}

	apiServer := NewServer(cfg)
	r := gin.Default()

	r.POST("/tenant/:tenantId/vm", apiServer.create())
	r.POST("/tenant/:tenantId/vm/:vmId", apiServer.delete())

	go func() {
		log.WithField("port", cfg.Port).Info("starting http server")
		if err := r.Run(); err != nil {
			log.WithError(err).Fatal("failed to start http server")
			os.Exit(1)
		}
	}()
	gracefulStop := make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)
	<-gracefulStop
	log.Info("shutting down server...")
}
