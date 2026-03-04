package api_server

import (
	"context"
	"fmt"
	"path"

	"github.com/edwin-Marrima/Tardigrade-runtime/pkg/config"
	"github.com/edwin-Marrima/Tardigrade-runtime/pkg/obs"
	frk "github.com/firecracker-microvm/firecracker-go-sdk"
	models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	log "github.com/sirupsen/logrus"
	"gvisor.dev/gvisor/pkg/cleanup"
)

type ApiServer struct {
	cfg *config.Config
}

func NewApiServer(cfg *config.Config) *ApiServer {
	return &ApiServer{
		cfg: cfg,
	}
}

func (as *ApiServer) Start(ctx context.Context, tenantId string, vm CreateVmRequest) (*CreateVmResponse, error) {
	key := fmt.Sprintf("%s.%s", tenantId, vm.Name)
	// create state dir,files
	cleanup := cleanup.Make(func() {
		log.WithFields(log.Fields{
			obs.TenantId: tenantId,
			obs.VmName:   vm.Name,
		}).Info("clean up vm creation Leftovers")
	})
	sockPath := fmt.Sprintf("%s.sock", key)
	c := frk.Config{
		SocketPath:      sockPath,
		InitrdPath:      as.cfg.Initramfs,
		KernelImagePath: as.cfg.Kernel,
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  &vm.ResourceAllocation.CpuCount,
			MemSizeMib: &vm.ResourceAllocation.MemorySizeMb,
		},
		NetworkInterfaces: frk.NetworkInterfaces{},
		//KernelArgs:
	}

	frk.NewMachine()
	// create logfile that will hold firecracker logs
	logFile := path.Join(as.cfg.StatePath, fmt.Sprintf("%s.log", key))
	// start firecracker process
	// allocate ip to virtual machine

	return nil, nil
}
