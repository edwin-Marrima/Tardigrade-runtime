
BUSYBOX_VERSION ?= 1.36.1
BUSYBOX_URL = https://busybox.net/downloads/binaries/$(BUSYBOX_VERSION)-x86_64-linux-musl/busybox

# CNI versions
CNI_PLUGINS_VERSION ?= 1.9.0
TC_REDIRECT_TAP_VERSION ?= 2024-02-14-1230
CNI_ARCH ?= amd64

BIN_DIR     := ./.bin
CNI_BIN_DIR := $(BIN_DIR)/cni
CNI_PLUGINS_URL := https://github.com/containernetworking/plugins/releases/download/v$(CNI_PLUGINS_VERSION)/cni-plugins-linux-$(CNI_ARCH)-v$(CNI_PLUGINS_VERSION).tgz
TC_REDIRECT_TAP_URL := https://github.com/alexellis/tc-tap-redirect-builder/releases/download/2024-02-14-1230/tc-redirect-tap

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

endef

export BIN_CONFIG_GO

$(BIN_DIR)/config.go: $(CNI_BIN_DIR)/ptp $(CNI_BIN_DIR)/host-local $(CNI_BIN_DIR)/tc-redirect-tap
	@echo "$$BIN_CONFIG_GO" > $@


busybox:
	curl -fsSL $(BUSYBOX_URL) -o $(BIN_DIR)/busybox
	chmod +x $(BIN_DIR)/busybox
