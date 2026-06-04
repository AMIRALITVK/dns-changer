package profiles

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Profile struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Servers []string `json:"servers"`
}

type Store struct {
	filePath string
}

func NewStore() (*Store, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.TempDir()
	}
	dir := filepath.Join(configDir, "dns-changer")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &Store{filePath: filepath.Join(dir, "profiles.json")}, nil
}

func (s *Store) Load() ([]Profile, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Profile{}, nil
		}
		return nil, err
	}
	var profiles []Profile
	if err := json.Unmarshal(data, &profiles); err != nil {
		return []Profile{}, nil
	}
	return profiles, nil
}

func (s *Store) Save(profiles []Profile) error {
	data, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}
