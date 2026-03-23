package app

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	api "github.com/edwin-Marrima/Tardigrade-runtime/pkg/api_server"
	"github.com/edwin-Marrima/Tardigrade-runtime/pkg/config"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type server struct {
	cfg       *config.Config
	apiServer *api.ApiServer
}

func newServer(cfg *config.Config) *server {
	return &server{
		cfg:       cfg,
		apiServer: api.NewApiServer(cfg),
	}
}

func (s *server) create() func(c *gin.Context) {
	return func(c *gin.Context) {
		tenantId := c.Param("tenantId")

		var req api.CreateVmRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := s.apiServer.Start(c.Request.Context(), tenantId, req)
		if err != nil {
			log.WithError(err).Error("failed to create vm")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, resp)
	}
}

func (s *server) delete() func(c *gin.Context) {
	return func(c *gin.Context) {
		tenantId := c.Param("tenantId")
		vmId := c.Param("vmId")

		if err := s.apiServer.Shutdown(c.Request.Context(), tenantId, vmId); err != nil {
			log.WithError(err).Error("failed to delete vm")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func NewServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the tardigrade runtime API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.NewConfig()
			if err != nil {
				log.WithError(err).Error("failed to load config")
				return fmt.Errorf("failed to load config: %w", err)
			}
			return runServer(cfg)
		},
	}

	return cmd
}

func runServer(cfg *config.Config) error {
	s := newServer(cfg)
	r := gin.Default()

	r.POST("/tenant/:tenantId/vm", s.create())
	r.DELETE("/tenant/:tenantId/vm/:vmId", s.delete())

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Port)
		log.WithField("port", cfg.Port).Info("starting http server")
		if err := r.Run(addr); err != nil {
			log.WithError(err).Fatal("failed to start http server")
			os.Exit(1)
		}
	}()

	gracefulStop := make(chan os.Signal, 1)
	signal.Notify(gracefulStop, syscall.SIGTERM, syscall.SIGINT)
	sig := <-gracefulStop
	log.WithField("signal", sig).Info("shutting down server")
	return nil
}
