package sysinfo

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Info struct {
	OS      string
	Version string
	Arch    string
	Kernel  string
	Shell   string
}

func Detect() Info {
	info := Info{
		OS:    runtime.GOOS,
		Arch:  runtime.GOARCH,
		Shell: os.Getenv("SHELL"),
	}
	info.Kernel = kernelRelease()

	switch runtime.GOOS {
	case "darwin":
		info.OS, info.Version = darwin()
	case "linux":
		info.OS, info.Version = linux()
	case "windows":
		info.OS, info.Version = windows()
	default:
		info.OS = runtime.GOOS
	}

	return info
}

func (i Info) Compact() string {
	osName := strings.TrimSpace(i.OS)
	if v := strings.TrimSpace(i.Version); v != "" {
		osName = strings.TrimSpace(osName + " " + v)
	}
	if osName == "" {
		osName = "unknown"
	}

	arch := i.Arch
	if arch == "" {
		arch = "unknown"
	}

	kernel := i.Kernel
	if kernel == "" {
		kernel = "unknown"
	}

	shell := filepath.Base(i.Shell)
	if shell == "" || shell == "." {
		shell = "unknown"
	}

	return fmt.Sprintf("%s · %s · %s · %s", osName, arch, kernel, shell)
}

func darwin() (name, version string) {
	name = "macOS"
	if out, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
		version = strings.TrimSpace(string(out))
	}
	if out, err := exec.Command("sw_vers", "-productName").Output(); err == nil {
		if n := strings.TrimSpace(string(out)); n != "" {
			name = n
		}
	}
	return name, version
}

func linux() (name, version string) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "Linux", kernelRelease()
	}
	defer f.Close()

	vals := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		i := strings.Index(line, "=")
		if i <= 0 {
			continue
		}
		vals[line[:i]] = strings.Trim(line[i+1:], `"`)
	}

	if v := vals["PRETTY_NAME"]; v != "" {
		return v, vals["VERSION_ID"]
	}
	if v := vals["NAME"]; v != "" {
		return v, vals["VERSION_ID"]
	}
	return "Linux", kernelRelease()
}

func windows() (name, version string) {
	out, err := exec.Command("cmd", "/c", "ver").Output()
	if err != nil {
		return "Windows", ""
	}
	return "Windows", strings.TrimSpace(string(out))
}

func kernelRelease() string {
	if runtime.GOOS == "windows" {
		return ""
	}
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
