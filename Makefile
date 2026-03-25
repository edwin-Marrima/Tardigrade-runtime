
BUSYBOX_VERSION ?= 1.36.1
BUSYBOX_URL = https://busybox.net/downloads/binaries/$(BUSYBOX_VERSION)-x86_64-linux-musl/busybox

# CNI versions
CNI_PLUGINS_VERSION ?= 1.9.0
TC_REDIRECT_TAP_VERSION ?= 2024-02-14-1230
LINUX_KERNEL_VERSION ?= 5.10.223
FIRECRACKER_VERSION ?= 1.10.1
TARGET_ARCH ?= arm64

# Map TARGET_ARCH to Firecracker's arch naming convention (arm64 -> aarch64, amd64 -> x86_64)
FIRECRACKER_ARCH := aarch64

BIN_DIR     := ./.bin
CNI_BIN_DIR := $(BIN_DIR)/cni
CNI_PLUGINS_URL := https://github.com/containernetworking/plugins/releases/download/v$(CNI_PLUGINS_VERSION)/cni-plugins-linux-$(TARGET_ARCH)-v$(CNI_PLUGINS_VERSION).tgz
TC_REDIRECT_TAP_URL := https://github.com/alexellis/tc-tap-redirect-builder/releases/download/2024-02-14-1230/tc-redirect-tap-arm64
LINUX_KERNEL_URL := https://s3.amazonaws.com/spec.ccfc.min/firecracker-ci/v1.10/$(TARGET_ARCH)/vmlinux-$(LINUX_KERNEL_VERSION)
FIRECRACKER_URL := https://github.com/firecracker-microvm/firecracker/releases/download/v$(FIRECRACKER_VERSION)/firecracker-v$(FIRECRACKER_VERSION)-$(FIRECRACKER_ARCH).tgz

PROTO_DIR    := proto
ROOTFS_IMAGE ?= tardigrade/rootfs
ROOTFS_TAG   ?= latest

.PHONY: rootfs-image
rootfs-image:
	docker build \
		-f rootfs/Dockerfile-rootfs \
		-t $(ROOTFS_IMAGE):$(ROOTFS_TAG) \
		.

.PHONY: proto
proto:
	protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/cmdserver.proto

.PHONY: vmlinux
vmlinux:
	curl -o $(BIN_DIR)/vmlinux -S -L \
	"$(LINUX_KERNEL_URL)"

.PHONY: cni-plugins
cni-plugins: $(CNI_BIN_DIR)/ptp $(CNI_BIN_DIR)/host-local $(CNI_BIN_DIR)/tc-redirect-tap $(BIN_DIR)/config.go

$(CNI_BIN_DIR)/ptp $(CNI_BIN_DIR)/host-local:
	mkdir -p $(CNI_BIN_DIR)
	curl -fsSL $(CNI_PLUGINS_URL) | tar -xz -C $(CNI_BIN_DIR) ./ptp ./host-local

$(CNI_BIN_DIR)/tc-redirect-tap:
	mkdir -p $(CNI_BIN_DIR)
	curl -fsSL $(TC_REDIRECT_TAP_URL) -o $(CNI_BIN_DIR)/tc-redirect-tap
	chmod +x $(CNI_BIN_DIR)/tc-redirect-tap

define BIN_CONFIG_GO
package _bin

import (
	_ "embed"
)

//go:embed cni/ptp
var CNIPtp []byte

//go:embed cni/host-local
var CNIHostLocal []byte

//go:embed cni/tc-redirect-tap
var CNITcRedirectTap []byte

//go:embed busybox
var Busybox []byte

//go:embed vmlinux
var Vmlinux []byte

//go:embed firecracker
var Firecracker []byte

endef

export BIN_CONFIG_GO

$(BIN_DIR)/config.go: $(CNI_BIN_DIR)/ptp $(CNI_BIN_DIR)/host-local $(CNI_BIN_DIR)/tc-redirect-tap $(BIN_DIR)/firecracker
	@echo "$$BIN_CONFIG_GO" > $@


busybox:
	curl -fsSL $(BUSYBOX_URL) -o $(BIN_DIR)/busybox
	chmod +x $(BIN_DIR)/busybox

.PHONY: firecracker
firecracker: $(BIN_DIR)/firecracker

$(BIN_DIR)/firecracker:
	mkdir -p $(BIN_DIR)
	curl -fsSL $(FIRECRACKER_URL) | tar -xz --strip-components=1 \
		-C $(BIN_DIR) \
		release-v$(FIRECRACKER_VERSION)-$(FIRECRACKER_ARCH)/firecracker-v$(FIRECRACKER_VERSION)-$(FIRECRACKER_ARCH)
	mv $(BIN_DIR)/firecracker-v$(FIRECRACKER_VERSION)-$(FIRECRACKER_ARCH) $(BIN_DIR)/firecracker
	chmod +x $(BIN_DIR)/firecracker

.PHONY: vagrant-sync
vagrant-sync:
	vagrant rsync-auto


setup:
	setup --rootfs="/tmp/rootfs.image" --cni-network-name=test --vm-cidr="10.10.1.10/24" --rootfs-image="tardigrade/rootfs:latest" --initramfs="/tmp/initramfs.cpio.gz"