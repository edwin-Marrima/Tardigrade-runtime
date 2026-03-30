package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	cfg "github.com/edwin-Marrima/Tardigrade-runtime/pkg/config"
)

func TestWriteCNIConf(t *testing.T) {
	config := &cfg.Config{
		Network: cfg.NetworkConfig{
			NetworkName: "tardigrade",
			Cidr:        "172.16.0.0/24",
		},
	}

	t.Run("writes conflist file with correct name", func(t *testing.T) {
		confDir := t.TempDir()

		if err := writeCNIConf(config, confDir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := filepath.Join(confDir, "tardigrade.conflist")
		if _, err := os.Stat(expected); err != nil {
			t.Errorf("expected conflist file at %s: %v", expected, err)
		}
	})

	t.Run("conflist has correct JSON structure", func(t *testing.T) {
		confDir := t.TempDir()

		if err := writeCNIConf(config, confDir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(filepath.Join(confDir, "tardigrade.conflist"))
		if err != nil {
			t.Fatalf("failed to read conflist: %v", err)
		}

		var conf map[string]any
		if err := json.Unmarshal(data, &conf); err != nil {
			t.Fatalf("conflist is not valid JSON: %v", err)
		}

		if conf["cniVersion"] != "1.0.0" {
			t.Errorf("cniVersion: got %v, want 1.0.0", conf["cniVersion"])
		}
		if conf["name"] != "tardigrade" {
			t.Errorf("name: got %v, want tardigrade", conf["name"])
		}

		plugins, ok := conf["plugins"].([]any)
		if !ok || len(plugins) != 2 {
			t.Fatalf("expected 2 plugins, got %v", conf["plugins"])
		}

		ptp := plugins[0].(map[string]any)
		if ptp["type"] != "ptp" {
			t.Errorf("first plugin type: got %v, want ptp", ptp["type"])
		}
		ipam := ptp["ipam"].(map[string]any)
		if ipam["type"] != "host-local" {
			t.Errorf("ipam type: got %v, want host-local", ipam["type"])
		}
		if ipam["subnet"] != "172.16.0.0/24" {
			t.Errorf("ipam subnet: got %v, want 172.16.0.0/24", ipam["subnet"])
		}

		tap := plugins[1].(map[string]any)
		if tap["type"] != "tc-redirect-tap" {
			t.Errorf("second plugin type: got %v, want tc-redirect-tap", tap["type"])
		}
	})

	t.Run("creates confDir if it does not exist", func(t *testing.T) {
		confDir := filepath.Join(t.TempDir(), "new", "nested", "dir")

		if err := writeCNIConf(config, confDir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(confDir); err != nil {
			t.Errorf("conf dir was not created: %v", err)
		}
	})
}

func TestSetupCNIBin(t *testing.T) {
	ptp := []byte("ptp-binary")
	hostLocal := []byte("host-local-binary")
	tcRedirectTap := []byte("tc-redirect-tap-binary")

	t.Run("writes all three plugin binaries", func(t *testing.T) {
		binDir := t.TempDir()

		if err := setupCNIBin(binDir, ptp, hostLocal, tcRedirectTap); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cases := []struct {
			name string
			want []byte
		}{
			{"ptp", ptp},
			{"host-local", hostLocal},
			{"tc-redirect-tap", tcRedirectTap},
		}
		for _, c := range cases {
			got, err := os.ReadFile(filepath.Join(binDir, c.name))
			if err != nil {
				t.Errorf("failed to read %s: %v", c.name, err)
				continue
			}
			if string(got) != string(c.want) {
				t.Errorf("%s content: got %q, want %q", c.name, got, c.want)
			}
		}
	})

	t.Run("binaries have executable permissions", func(t *testing.T) {
		binDir := t.TempDir()

		if err := setupCNIBin(binDir, ptp, hostLocal, tcRedirectTap); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, name := range []string{"ptp", "host-local", "tc-redirect-tap"} {
			info, err := os.Stat(filepath.Join(binDir, name))
			if err != nil {
				t.Errorf("failed to stat %s: %v", name, err)
				continue
			}
			if info.Mode()&0111 == 0 {
				t.Errorf("%s: expected executable permissions, got %v", name, info.Mode())
			}
		}
	})

	t.Run("creates binDir if it does not exist", func(t *testing.T) {
		binDir := filepath.Join(t.TempDir(), "opt", "cni", "bin")

		if err := setupCNIBin(binDir, ptp, hostLocal, tcRedirectTap); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(binDir); err != nil {
			t.Errorf("bin dir was not created: %v", err)
		}
	})
}
