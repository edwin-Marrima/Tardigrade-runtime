package api_server

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"

	"github.com/edwin-Marrima/Tardigrade-runtime/pkg/config"
	"github.com/edwin-Marrima/Tardigrade-runtime/pkg/network"
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
	key := path.Join(as.cfg.StatePath, fmt.Sprintf("%s.%s", tenantId, vm.Name))
	// create state dir
	if err := os.MkdirAll(key, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	cu := cleanup.Make(func() {
		log.WithFields(log.Fields{
			obs.TenantId: tenantId,
			obs.VmName:   vm.Name,
		}).Info("clean up vm creation Leftovers")
	})
	cu.Add(func() {
		_ = os.RemoveAll(key)
	})
	defer cu.Clean()

	sockPath := path.Join(key, "vm.sock")
	logFile := path.Join(key, "vm.log")
	ext4Fs := path.Join(key, "vm.ext4")
	if err := makeWritableFS(ctx, ext4Fs, vm.ResourceAllocation.DiskSizeMb); err != nil {
		return nil, fmt.Errorf("failed to create ext4 filesystem: %w", err)
	}
	vmCidrInfo, err := network.ParseCIDR(as.cfg.VMCidr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse micro vm cidr: %w", err)
	}
	netMask := net.IP(vmCidrInfo.Mask).String()
	cfg := frk.Config{
		LogPath:         logFile,
		SocketPath:      sockPath,
		InitrdPath:      as.cfg.Initramfs,
		KernelImagePath: as.cfg.Kernel,
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  &vm.ResourceAllocation.CpuCount,
			MemSizeMib: &vm.ResourceAllocation.MemorySizeMb,
		},
		NetworkInterfaces: frk.NetworkInterfaces{
			{CNIConfiguration: &frk.CNIConfiguration{
				NetworkName: as.cfg.CNINetworkName,
				IfName:      "veth0",
			}},
		},
		Drives: []models.Drive{
			{
				DriveID:      toPointer("rootfs"),
				IsReadOnly:   toPointer(true),
				IsRootDevice: toPointer(false),
				PathOnHost:   toPointer(as.cfg.Rootfs),
			},
			{
				DriveID:      toPointer("overlayfs"),
				IsReadOnly:   toPointer(false),
				PathOnHost:   toPointer(ext4Fs),
				IsRootDevice: toPointer(false),
			},
		},
		KernelArgs: fmt.Sprintf("console=ttyS0 reboot=k panic=1  pci=off init=/init ip=::%s:%s::eth0:off", vmCidrInfo.Gateway.String(), netMask),
	}

	cmd := frk.VMCommandBuilder{}.
		WithBin(as.cfg.FireCrackerBinPath).Build(ctx)
	m, err := frk.NewMachine(ctx, cfg, frk.WithProcessRunner(cmd))
	if err != nil {
		return nil, fmt.Errorf("failed to create micro vm: %w", err)
	}

	if err := m.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start micro vm: %w", err)
	}

	// Extract the tap device name and IP address populated by the SDK
	// from the CNI result after start.
	iface := m.Cfg.NetworkInterfaces[0]
	var ipAddr, tapName string
	if iface.StaticConfiguration != nil {
		tapName = iface.StaticConfiguration.HostDevName
		if iface.StaticConfiguration.IPConfiguration != nil {
			ipAddr = iface.StaticConfiguration.IPConfiguration.IPAddr.IP.String()
		}
	}

	cu.Release()

	return &CreateVmResponse{
		Name:               vm.Name,
		ResourceAllocation: vm.ResourceAllocation,
		NetworkDeviceName:  tapName,
		IpAddress:          ipAddr,
	}, nil
}

func makeWritableFS(ctx context.Context, imgPath string, sizeInMbs int64) error {
	logger := log.WithFields(log.Fields{
		"sizeInMbs": sizeInMbs,
		"path":      imgPath,
	})
	logger.Info("writing writable filesystem")

	ddCmd := exec.CommandContext(ctx, "dd",
		"if=/dev/zero",
		fmt.Sprintf("of=%s", imgPath),
		"bs=1M",
		fmt.Sprintf("count=%d", sizeInMbs))
	if out, err := ddCmd.CombinedOutput(); err != nil {
		logger.WithError(err).Error("failed to create blank disk image")
		return fmt.Errorf("failed to create blank disk image at %s: %w, output: %s", imgPath, err, out)
	}

	mkfsCmd := exec.CommandContext(ctx, "mkfs.ext4", imgPath)
	if out, err := mkfsCmd.CombinedOutput(); err != nil {
		logger.WithError(err).Error("failed to format disk image as ext4")
		return fmt.Errorf("failed to format %s as ext4: %w, output: %s", imgPath, err, out)
	}
	return nil
}

func toPointer[T any](s T) *T {
	return &s
}
