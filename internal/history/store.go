package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const maxEntries = 1000

type Entry struct {
	Prompt   string    `json:"prompt"`
	Response string    `json:"response"`
	Time     time.Time `json:"time"`
}

type Store struct {
	mu      sync.Mutex
	path    string
	entries []Entry
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ai-cli", "history.json"), nil
}

func Load(path string) (*Store, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}

	s := &Store{path: path}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, &s.entries); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

func (s *Store) Entries() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out
}

func (s *Store) Delete(index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if index < 0 || index >= len(s.entries) {
		return nil
	}

	s.entries = append(s.entries[:index], s.entries[index+1:]...)
	return s.saveLocked()
}

func (s *Store) Add(prompt, response string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries = append([]Entry{{
		Prompt:   prompt,
		Response: response,
		Time:     time.Now(),
	}}, s.entries...)

	if len(s.entries) > maxEntries {
		s.entries = s.entries[:maxEntries]
	}

	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s.entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}
