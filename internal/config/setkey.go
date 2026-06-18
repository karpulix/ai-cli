package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

func SetKeyInteractive() error {
	in, err := inputTTY()
	if err != nil {
		return err
	}
	defer in.Close()

	fmt.Fprint(os.Stderr, "OpenAI API key: ")

	key, err := readSecret(in)
	if err != nil {
		return err
	}
	if key == "" {
		return fmt.Errorf("empty API key")
	}

	cfg, err := Load()
	if err != nil {
		return err
	}
	if err := cfg.Upsert("default", Profile{APIKey: key}); err != nil {
		return err
	}
	cfg.ActiveProfile = "default"
	if err := cfg.Save(); err != nil {
		return err
	}
	path, _ := Path()
	fmt.Fprintf(os.Stderr, "saved to %s (mode 600)\n", path)
	return nil
}

func readSecret(in io.Reader) (string, error) {
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(b)), nil
	}
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func inputTTY() (io.ReadCloser, error) {
	if term.IsTerminal(int(syscall.Stdin)) {
		return nopCloser{os.Stdin}, nil
	}
	return os.OpenFile("/dev/tty", os.O_RDWR, 0)
}

type nopCloser struct{ io.Reader }

func (nopCloser) Close() error { return nil }
