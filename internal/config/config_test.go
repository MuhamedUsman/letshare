package config

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestGet(t *testing.T) {
	// get the prev state that we'll restore
	prev, err := Get()
	if err != nil {
		// if not exists, it must create the config with defaults
		if errors.Is(err, ErrNoConfig) {
			prev, err = Load()
		}
		assert.NotErrorIs(t, err, ErrNoConfig, "failed to get/load config, got: %v", err)
	}
	// defer the call to restore the previous state
	defer func() {
		err := Save(prev)
		assert.NoErrorf(t, err, "failed to restore previous config: %v", err)
	}()
	// now get the file and delete it
	f, err := getUserConfigFile()
	assert.NoErrorf(t, err, "failed to get user config file: %v", err)
	assert.NoError(t, f.Close(), "failed to close user config file: %v", err)
	// remove the file
	assert.NoErrorf(t, os.Remove(f.Name()), "failed to remove user config file: %v", err)

	// now save a new config
	cfg := Config{
		Personal: PersonalConfig{
			Username: "TestUser",
		},
		Share: ShareConfig{
			InstanceName:      "TestInstance",
			StoppableInstance: true,
			ZipFiles:          true,
			Compression:       true,
			SharedZipName:     "Test.zip",
		},
		Receive: ReceiveConfig{
			DownloadFolder:      "testPath",
			ConcurrentDownloads: 0,
		},
	}

	// save the config
	assert.NoErrorf(t, Save(cfg), "failed to save config: %v", err)

	// now get the config again
	// it must be loaded as Save() method will load the saved config
	saved, err := Get()
	assert.NoErrorf(t, err, "failed to get config: %v", err)
	assert.Exactly(t, cfg, saved, "Saved config does not match expected config")
}
