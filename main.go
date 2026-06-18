package main

import (
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/karpulix/ai-cli/internal/app"
	"github.com/karpulix/ai-cli/internal/config"
	"github.com/karpulix/ai-cli/internal/history"
	"github.com/karpulix/ai-cli/internal/llm"
	"github.com/karpulix/ai-cli/internal/shell"
	"github.com/karpulix/ai-cli/internal/version"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-version", "version":
			fmt.Println(version.String())
			return
		case "init":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: ai-cli init <zsh|bash>")
				os.Exit(1)
			}
			bin, err := os.Executable()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			switch os.Args[2] {
			case "zsh":
				fmt.Print(shell.InitZsh(bin))
			case "bash":
				fmt.Print(shell.InitBash(bin))
			default:
				fmt.Fprintln(os.Stderr, "usage: ai-cli init <zsh|bash>")
				os.Exit(1)
			}
			return
		case "install":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: ai-cli install <zsh|bash>")
				os.Exit(1)
			}
			sh := os.Args[2]
			if err := shell.Install(sh); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			rc := map[string]string{"zsh": "~/.zshrc", "bash": "~/.bashrc"}[sh]
			fmt.Printf("Ctrl+A binding added to %s\n", rc)
			fmt.Println("Run: source " + rc)
			return
		case "config":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "usage: ai-cli config set-key")
				os.Exit(1)
			}
			switch os.Args[2] {
			case "set-key":
				if err := config.SetKeyInteractive(); err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
			case "refresh-prompt":
				cfg, err := config.Load()
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				if err := cfg.ResetPromptTemplate(); err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				fmt.Println("prompt template reset to default")
			default:
				fmt.Fprintln(os.Stderr, "usage: ai-cli config <set-key|refresh-prompt>")
				os.Exit(1)
			}
			return
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	store, err := history.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "history: %v\n", err)
		os.Exit(1)
	}

	var client *llm.Client
	if cfg.HasProfiles() {
		client, _ = llm.New()
	}

	openMaster := !cfg.HasProfiles()

	shellMode := !isTTY(os.Stdout)

	output := io.Writer(os.Stdout)
	opts := []tea.ProgramOption{tea.WithAltScreen()}
	if shellMode {
		tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "tty: %v\n", err)
			os.Exit(1)
		}
		defer tty.Close()
		output = tty
		opts = append(opts, tea.WithInput(tty), tea.WithOutput(tty))
	}

	app.SetupRenderer(output)

	m := app.New(store, cfg, client, openMaster)
	program := tea.NewProgram(m, opts...)

	final, err := program.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	result := final.(app.Model).Result()
	if result == "" {
		return
	}

	if shellMode {
		fmt.Print(result)
		return
	}

	if err := shell.WriteResult(result); err != nil {
		fmt.Fprintf(os.Stderr, "output: %v\n", err)
		os.Exit(1)
	}
}

func isTTY(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
