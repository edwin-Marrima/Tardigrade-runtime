//go:build ignore

package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
)

const (
	defaultHostname = "tardigrade"
)

func main() {
	setHostname()
	setupDNS()
}

func setupDNS() {
	if err := os.WriteFile("/etc/resolv.conf", []byte("nameserver 1.1.1.1\n"), 0644); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to write /etc/resolv.conf: %v\n", err)
	}

}
func setHostname() {
	hostname := defaultHostname
	params, err := ParseKernelCmdline()
	if err == nil {
		var ok bool
		hostname, ok = params["hostname"]
		if !ok {
			_, _ = fmt.Fprintf(os.Stderr, "hostname not provided in kernel cmdline, using default")
		}
	}

	if _, err := os.Hostname(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "warning: could not get hostname: %v\n", err)
	}

	if err := syscall.Sethostname([]byte(hostname)); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to set hostname: %v\n", err)
	}
}
func ParseKernelCmdline() (map[string]string, error) {
	data, err := os.ReadFile("/proc/cmdline")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc/cmdline: %w", err)
	}

	params := make(map[string]string)
	for _, field := range strings.Fields(strings.TrimSpace(string(data))) {
		key, value, found := strings.Cut(field, "=")
		if found {
			params[key] = value
		} else {
			params[key] = ""
		}
	}
	return params, nil
}
