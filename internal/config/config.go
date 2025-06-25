package config

import (
	"errors"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const (
	appConfDir  = ".letshare"
	testConfDir = ".test"
	appConfFile = "config.toml"
)

var (
	ErrNoConfig = errors.New("config must be loaded")
)

var TestFlag bool

func init() {
	flag.BoolVar(&TestFlag,
		"test",
		false,
		"Use to run app with separate user config file, useful for testing purposes",
	)
	// parsed in main package
}

type PersonalConfig struct {
	Username string `toml:"username"`
}

type ShareConfig struct {
	InstanceName      string `toml:"instance_name"`
	StoppableInstance bool   `toml:"stoppable_instance"`
	ZipFiles          bool   `toml:"zip_files"`
	Compression       bool   `toml:"compression"`
	SharedZipName     string `toml:"shared_zip_name"`
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

// Get returns the lastest loaded/saved user's config,
// if it returns ErrNoConfig, Load OR Save must be called.
func Get() (Config, error) {
	mu.Lock()
	defer mu.Unlock()
	if config != nil {
		return *config, nil
	}
	return Config{}, ErrNoConfig
}

// Load loads the configuration from the user's config file.
func Load() (Config, error) {
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

// Save saves the configuration to the user's config file.
func Save(c Config) error {
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
			InstanceName:      hostname,
			StoppableInstance: true,
			ZipFiles:          false,
			SharedZipName:     "shared.zip",
		},
		Receive: ReceiveConfig{
			DownloadFolder: downPath,
		},
	}
	return cfg, nil
}

func getUserConfigFile() (*os.File, error) {
	d, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("user config directory look-up: %w", err)
	}

	path := filepath.Join(d, appConfDir, appConfFile)
	if TestFlag {
		path = filepath.Join(d, testConfDir, appConfFile)
	}
	var f *os.File
	if f, err = os.Open(path); err != nil {
		return nil, fmt.Errorf("opening app config file: %w", err)
	}
	return f, nil
}

func createConfigFile() (*os.File, error) {
	d, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("user config directory look-up: %v", err)
	}

	path := filepath.Join(d, appConfDir)
	if TestFlag {
		path = filepath.Join(d, testConfDir)
	}
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
