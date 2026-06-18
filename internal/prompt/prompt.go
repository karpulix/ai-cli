package prompt

import (
	"fmt"
	"strings"

	"github.com/karpulix/ai-cli/internal/sysinfo"
)

const SystemInfoPlaceholder = "{system_info}"

const DefaultTemplate = `Shell command assistant. You understand the conventions, CLI tools, and quirks of the user's operating system.

Target environment:
` + SystemInfoPlaceholder + `

Return only the command(s) to run — no markdown or explanation. Join multiple with &&.`

func Default() string {
	return DefaultTemplate
}

func HasPlaceholder(template string) bool {
	return strings.Contains(template, SystemInfoPlaceholder)
}

func FormatSystemInfo(info sysinfo.Info) string {
	shell := info.Shell
	if shell == "" {
		shell = "unknown"
	}
	version := info.Version
	if version == "" {
		version = "unknown"
	}
	kernel := info.Kernel
	if kernel == "" {
		kernel = "unknown"
	}

	return fmt.Sprintf(`- OS: %s %s
- Architecture: %s
- Kernel: %s
- Default shell: %s`, info.OS, version, info.Arch, kernel, shell)
}

func MigrateTemplate(template string) string {
	if HasPlaceholder(template) {
		return template
	}

	for _, header := range []string{"Target environment:", "Runtime environment:"} {
		idx := strings.Index(template, header)
		if idx < 0 {
			continue
		}
		after := template[idx+len(header):]
		lines := strings.Split(after, "\n")
		end := 0
		for end < len(lines) {
			line := strings.TrimSpace(lines[end])
			if line == "" {
				end++
				break
			}
			if strings.HasPrefix(line, "- ") {
				end++
				continue
			}
			break
		}
		rest := strings.TrimLeft(strings.Join(lines[end:], "\n"), "\n")
		before := strings.TrimRight(template[:idx+len(header)], " \t")
		return before + "\n" + SystemInfoPlaceholder + "\n\n" + rest
	}

	return DefaultTemplate
}

func Render(template string, info sysinfo.Info) string {
	template = strings.TrimSpace(template)
	if template == "" {
		template = DefaultTemplate
	}
	if !HasPlaceholder(template) {
		template = MigrateTemplate(template)
	}
	return strings.ReplaceAll(template, SystemInfoPlaceholder, FormatSystemInfo(info))
}

func RenderNow(template string) string {
	return Render(template, sysinfo.Detect())
}
