package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const zshMarker = "# ai-cli shell integration"

func InitZsh(bin string) string {
	return fmt.Sprintf(`%s
_ai_cli() {
  emulate -L zsh
  setopt local_options no_aliases

  zle -I

  local result
  result=$('%s' < /dev/tty 2>/dev/tty)
  stty sane 2>/dev/null

  if [[ -n "$result" ]]; then
    LBUFFER="${result}${LBUFFER}"
    CURSOR=${#result}
  fi

  zle reset-prompt
}
zle -N _ai_cli
bindkey '^A' _ai_cli
`, zshMarker, bin)
}

func InitBash(bin string) string {
	return fmt.Sprintf(`%s
_ai_cli() {
  local result
  result=$('%s' < /dev/tty 2>/dev/tty)
  stty sane 2>/dev/null
  if [[ -n "$result" ]]; then
    READLINE_LINE="${result}${READLINE_LINE}"
    READLINE_POINT=${#result}
  fi
}
bind -x '"\C-a": _ai_cli'
`, zshMarker, bin)
}

var snippetEnd = regexp.MustCompile(`(?m)^bindkey '\^A' _ai_cli$|^bind -x '"\\C-a": _ai_cli'$`)

func replaceSnippet(content, snippet string) string {
	start := strings.Index(content, zshMarker)
	if start == -1 {
		if len(content) > 0 && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		return content + "\n" + snippet
	}

	rest := content[start:]
	loc := snippetEnd.FindStringIndex(rest)
	if loc == nil {
		return content[:start] + snippet
	}

	end := start + loc[1]
	if end < len(content) && content[end] == '\n' {
		end++
	}
	return content[:start] + snippet + content[end:]
}

func Install(shellName string) error {
	bin, err := os.Executable()
	if err != nil {
		return err
	}
	bin, err = filepath.EvalSymlinks(bin)
	if err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	var rcPath string
	var snippet string

	switch shellName {
	case "zsh":
		rcPath = filepath.Join(home, ".zshrc")
		snippet = InitZsh(bin)
	case "bash":
		rcPath = filepath.Join(home, ".bashrc")
		snippet = InitBash(bin)
	default:
		return fmt.Errorf("unsupported shell: %s (use zsh or bash)", shellName)
	}

	data, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	content := string(data)
	content = replaceSnippet(content, snippet)

	return os.WriteFile(rcPath, []byte(content), 0o644)
}
