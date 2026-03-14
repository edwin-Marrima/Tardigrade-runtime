package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	bin "github.com/edwin-Marrima/Tardigrade-runtime/.bin"
	cfg "github.com/edwin-Marrima/Tardigrade-runtime/pkg/config"
	"github.com/edwin-Marrima/Tardigrade-runtime/pkg/utils"
	log "github.com/sirupsen/logrus"
)

const cniBinDir = "/opt/cni/bin"
const cniConfDir = "/etc/cni/net.d"

func Cni(c *cfg.Config) error {
	if err := writeCNIConf(c, cniConfDir); err != nil {
		return err
	}
	return setupCNIBin(cniBinDir, bin.CNIPtp, bin.CNIHostLocal, bin.CNITcRedirectTap)
}

func writeCNIConf(c *cfg.Config, confDir string) error {
	if err := os.MkdirAll(confDir, 0755); err != nil {
		return fmt.Errorf("failed to create CNI conf dir: %w", err)
	}

	conf := map[string]any{
		"cniVersion": "1.0.0",
		"name":       c.CNINetworkName,
		"plugins": []map[string]any{
			{
				"type":   "ptp",
				"ipMasq": true,
				"ipam": map[string]any{
					"type":   "host-local",
					"subnet": c.VMCidr,
				},
			},
			{
				"type": "tc-redirect-tap",
			},
		},
	}

	data, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal CNI config: %w", err)
	}

	confPath := filepath.Join(confDir, c.CNINetworkName+".conflist")
	if err := os.WriteFile(confPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write CNI config: %w", err)
	}

	log.Debugf("CNI config written to %s", confPath)
	return nil
}

// setupCNIBin installs the provided CNI plugin binaries to binDir.
func setupCNIBin(binDir string, ptp, hostLocal, tcRedirectTap []byte) error {
	plugins := []struct {
		name string
		data []byte
	}{
		{"ptp", ptp},
		{"host-local", hostLocal},
		{"tc-redirect-tap", tcRedirectTap},
	}

	for _, p := range plugins {
		path := filepath.Join(binDir, p.name)
		if err := utils.WriteFile(p.data, path); err != nil {
			return fmt.Errorf("failed to install CNI plugin %s: %w", p.name, err)
		}
		log.Infof("installed CNI plugin: %s", path)
	}

	return nil
}
