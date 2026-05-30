package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"ibkr-stock-analysis/internal/domain"
)

type Snapshot struct {
	Settings domain.Settings         `json:"settings"`
	History  []domain.AnalysisResult `json:"history"`
}

type Store struct {
	path         string
	historyLimit int
}

func NewStore(path string, historyLimit int) *Store {
	if historyLimit <= 0 {
		historyLimit = 50
	}
	return &Store{path: path, historyLimit: historyLimit}
}

func DefaultPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "ibkr-stock-analysis", "settings.json"), nil
}

func (s *Store) Load() (Snapshot, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Snapshot{Settings: domain.DefaultSettings(), History: []domain.AnalysisResult{}}, nil
		}
		return Snapshot{}, err
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{Settings: domain.DefaultSettings(), History: []domain.AnalysisResult{}}, nil
	}
	snapshot.Settings = snapshot.Settings.Normalize()
	if snapshot.History == nil {
		snapshot.History = []domain.AnalysisResult{}
	}
	return s.trim(snapshot), nil
}

func (s *Store) Save(snapshot Snapshot) error {
	snapshot.Settings = snapshot.Settings.Normalize()
	snapshot = s.trim(snapshot)
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

func (s *Store) trim(snapshot Snapshot) Snapshot {
	if snapshot.History == nil {
		snapshot.History = []domain.AnalysisResult{}
	}
	if len(snapshot.History) > s.historyLimit {
		snapshot.History = snapshot.History[len(snapshot.History)-s.historyLimit:]
	}
	return snapshot
}
