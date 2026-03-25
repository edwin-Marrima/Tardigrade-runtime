package api_server

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"

	"github.com/edwin-Marrima/Tardigrade-runtime/pkg/config"
	"github.com/edwin-Marrima/Tardigrade-runtime/pkg/network"
	"github.com/edwin-Marrima/Tardigrade-runtime/pkg/obs"
	frk "github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	log "github.com/sirupsen/logrus"
	"gvisor.dev/gvisor/pkg/cleanup"
)

const (
	vethNetNsPairName = "veth0"

	cniNetNSDir = "/var/run/netns"
	cniCacheDir = "/var/lib/cni"
	cniConfDir  = "/etc/cni/conf.d"
	cniBinDir   = "/opt/cni/bin"
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
	ln := log.WithFields(log.Fields{
		obs.TenantId: tenantId,
		obs.VmName:   vm.Name,
		obs.Action:   "start",
	})
	key := path.Join(as.cfg.Runtime.StateFolder, fmt.Sprintf("%s.%s", tenantId, vm.Name))
	// create state dir
	if err := os.MkdirAll(key, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	cu := cleanup.Make(func() {
		ln.Info("clean up vm creation Leftovers")
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
	vmCidrInfo, err := network.ParseCIDR(as.cfg.Network.Cidr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse micro vm cidr: %w", err)
	}
	netMask := net.IP(vmCidrInfo.Mask).String()
	cfg := frk.Config{
		VMID:            fmt.Sprintf("%s-%s", tenantId, vm.Name),
		LogPath:         logFile,
		SocketPath:      sockPath,
		InitrdPath:      as.cfg.Filesystem.InitramPath,
		KernelImagePath: as.cfg.Runtime.LinuxKernelPath,
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  &vm.ResourceAllocation.CpuCount,
			MemSizeMib: &vm.ResourceAllocation.MemorySizeMb,
		},
		NetworkInterfaces: frk.NetworkInterfaces{
			{CNIConfiguration: &frk.CNIConfiguration{
				NetworkName: as.cfg.Network.NetworkName,
				IfName:      vethNetNsPairName,
			}},
		},
		Drives: []models.Drive{
			{
				DriveID:      toPointer("rootfs"),
				IsReadOnly:   toPointer(true),
				IsRootDevice: toPointer(false),
				PathOnHost:   toPointer(as.cfg.Filesystem.Rootfs.Path),
			},
			{
				DriveID:      toPointer("overlayfs"),
				IsReadOnly:   toPointer(false),
				PathOnHost:   toPointer(ext4Fs),
				IsRootDevice: toPointer(false),
			},
		},
		//KernelArgs: fmt.Sprintf("console=ttyS0 reboot=k panic=1  pci=off init=/init ip=::%s:%s::eth0:off hostname=%s", vmCidrInfo.Gateway.String(), netMask, vm.Name),
		KernelArgs: fmt.Sprintf("console=ttyS0 reboot=k panic=1  pci=off init=/init  hostname=%s", vm.Name),
	}
	ln.WithFields(log.Fields{
		"vm.id":             cfg.VMID,
		"socket.apth":       cfg.SocketPath,
		"init.ram.path":     cfg.InitrdPath,
		"kernel.image.path": cfg.KernelImagePath,
		"vcpu":              cfg.MachineCfg.VcpuCount,
		"memory":            cfg.MachineCfg.MemSizeMib,
		"gateway":           vmCidrInfo.Gateway.String(),
		"network.mask":      netMask,
		"number.of.drives":  len(cfg.Drives),
		"network.ifname":    vethNetNsPairName,
		"network.cni.name":  as.cfg.Network.NetworkName,
	}).Info("starting VM")
	cmd := frk.VMCommandBuilder{}.
		WithBin(as.cfg.Runtime.FirecrackerPath).
		WithSocketPath(sockPath).
		Build(context.Background())
	//#TODO: replace with logger on cfg
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	m, err := frk.NewMachine(ctx, cfg, frk.WithProcessRunner(cmd))
	if err != nil {
		return nil, fmt.Errorf("failed to create micro vm: %w", err)
	}
	if err := m.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start micro vm: %w", err)
	}
	vmPID, err := m.PID()
	if err != nil {
		return nil, fmt.Errorf("failed to get micro vm pid: %w", err)
	}
	pidFile := path.Join(key, "vm.pid")
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(vmPID)), 0600); err != nil {
		return nil, fmt.Errorf("failed to write pid file: %w", err)
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

func (as *ApiServer) Shutdown(ctx context.Context, tenantId, vmName string) error {
	logger := log.WithFields(log.Fields{
		obs.TenantId: tenantId,
		obs.VmName:   vmName,
		obs.Action:   "shutdown",
	})

	key := path.Join(as.cfg.Runtime.StateFolder, fmt.Sprintf("%s.%s", tenantId, vmName))
	sockPath := path.Join(key, "vm.sock")
	cfg := frk.Config{
		VMID:       fmt.Sprintf("%s-%s", tenantId, vmName),
		SocketPath: sockPath,
	}

	logger.Info("connecting to micro vm")
	m, err := frk.NewMachine(ctx, cfg)
	if err != nil {
		logger.WithError(err).Error("failed to connect to micro vm")
		return fmt.Errorf("failed to shutdown micro vm: %w", err)
	}

	logger.Info("sending shutdown signal to micro vm")
	if err := m.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("failed to shutdown micro vm")
		return fmt.Errorf("failed to shutdown micro vm: %w", err)
	}

	logger.WithField("path", key).Info("removing micro vm state directory")
	if err := os.RemoveAll(key); err != nil {
		logger.WithError(err).Error("failed to remove micro vm state directory")
		return fmt.Errorf("failed to remove %s: %w", key, err)
	}

	logger.Info("micro vm shutdown complete")
	return nil
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
