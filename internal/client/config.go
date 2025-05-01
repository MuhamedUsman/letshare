package client

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	appConfDir  = ".letshare"
	appConfFile = "config.toml"
)

var (
	ErrNoConfig = errors.New("config must be loaded")
)

type PersonalConfig struct {
	Username string `toml:"username"`
}

type ShareConfig struct {
	ZipFiles      bool   `toml:"zip_files"`
	IsolateFiles  bool   `toml:"isolate_files"`
	SharedZipName string `toml:"shared_zip_name"`
}

type ReceiveConfig struct {
	DownloadFolder string `toml:"download_folder"`
}

type Config struct {
	Personal PersonalConfig `toml:"personal"`
	Share    ShareConfig    `toml:"share"`
	Receive  ReceiveConfig  `toml:"receive"`
}

var (
	mu     sync.Mutex
	config *Config
)

// GetConfig returns the lastest loaded/saved user's config,
// if it returns ErrNoConfig, LoadConfig OR SaveConfig must be called.
func GetConfig() (Config, error) {
	mu.Lock()
	defer mu.Unlock()
	if config != nil {
		return *config, nil
	}
	return Config{}, ErrNoConfig
}

// LoadConfig loads the configuration from the user's config file.
func LoadConfig() (Config, error) {
	f, err := getUserConfigFile()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			f, err = createConfigFile()
			if err != nil {
				return Config{}, fmt.Errorf("config file not exists, creating config file: %w", err)
			}
			defer f.Close()

			var cfg Config
			if cfg, err = defaultConfig(); err != nil {
				return Config{}, fmt.Errorf("getting default config: %w", err)
			}

			if err = writeConfig(f, cfg); err != nil {
				return Config{}, fmt.Errorf("writing default config to app config file: %w", err)
			}
			return cfg, nil
		} else {
			return Config{}, fmt.Errorf("opening config file: %w", err)
		}
	}
	defer f.Close()

	cfg, err := readConfig(f)
	if err != nil {
		return Config{}, err
	}
	// update config
	mu.Lock()
	defer mu.Unlock()
	config = &cfg

	return cfg, nil
}

// SaveConfig saves the configuration to the user's config file.
func SaveConfig(c Config) error {
	f, err := createConfigFile()
	if err != nil {
		return fmt.Errorf("creating/truncating config file: %w", err)
	}
	defer f.Close()
	if err = writeConfig(f, c); err != nil {
		return fmt.Errorf("writing new config to file: %w", err)
	}
	// update config
	mu.Lock()
	defer mu.Unlock()
	config = &c

	return nil
}

func defaultConfig() (Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("user home directory look-up: %w", err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		return Config{}, fmt.Errorf("hostname look-up: %w", err)
	}
	downPath := filepath.Join(homeDir, "Downloads")
	downPath = filepath.ToSlash(downPath)
	cfg := Config{
		Personal: PersonalConfig{
			Username: hostname,
		},
		Share: ShareConfig{
			ZipFiles:      false,
			IsolateFiles:  false,
			SharedZipName: "shared.zip",
		},
		Receive: ReceiveConfig{
			DownloadFolder: downPath,
		},
	}
	return cfg, nil
}

func getUserConfigFile() (*os.File, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("user config directory look-up: %w", err)
	}

	path := filepath.Join(dir, appConfDir, appConfFile)
	var f *os.File
	if f, err = os.Open(path); err != nil {
		return nil, fmt.Errorf("opening app config file: %w", err)
	}
	return f, nil
}

func createConfigFile() (*os.File, error) {
	ucd, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("user config directory look-up: %v", err)
	}

	path := filepath.Join(ucd, appConfDir)
	if err = os.MkdirAll(path, 0o700); err != nil {
		return nil, fmt.Errorf("creating app config directory: %w", err)
	}

	path = filepath.Join(path, appConfFile)
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("creating app config file: %w", err)
	}
	return f, nil
}

func readConfig(r io.Reader) (Config, error) {
	cfg := new(Config)
	if _, err := toml.NewDecoder(r).Decode(cfg); err != nil {
		return Config{}, fmt.Errorf("decoding config file: %w", err)
	}
	return *cfg, nil
}

func writeConfig(w io.Writer, c Config) error {
	if err := toml.NewEncoder(w).Encode(c); err != nil {
		return fmt.Errorf("encoding config file: %w", err)
	}
	return nil
}
