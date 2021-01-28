package cli

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/pelletier/go-toml"
)

type Config struct {
	Offset    uint   `toml:"offset"`
	Limit     uint   `toml:"limit"`
	Name      string `toml:"name"`
	RawOutput bool   `toml:"raw_output"`
}

// save - store config in a file
func save(c Config, file string) error {
	b, err := toml.Marshal(c)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to read config file: %s", err))
	}
	if err := ioutil.WriteFile(file, b, 0644); err != nil {
		return errors.New(fmt.Sprintf("failed to write config TOML: %s", err))
	}
	return nil
}

// read - retrieve config from a file
func read(file string) (Config, error) {
	data, err := ioutil.ReadFile(file)
	c := Config{}
	if err != nil {
		return c, errors.New(fmt.Sprintf("failed to read config file: %s", err))
	}

	if err := toml.Unmarshal(data, &c); err != nil {
		return Config{}, errors.New(fmt.Sprintf("failed to unmarshal config TOML: %s", err))
	}
	return c, nil
}

func getConfigPath() (string, error) {
	// Check if a config path passed by user exists.
	if ConfigPath != "" {
		if _, err := os.Stat(ConfigPath); os.IsNotExist(err) {
			errConfigNotFound := errors.Wrap(errors.New("config file was not found"), err)
			logError(errConfigNotFound)
			return "", err
		}
	}

	// If not, then read it from the user config directory.
	if ConfigPath == "" {
		userConfigDir, _ := os.UserConfigDir()
		ConfigPath = path.Join(userConfigDir, "mainflux", "cli.toml")
	}

	if _, err := os.Stat(ConfigPath); os.IsNotExist(err) {
		return "", err
	}

	return ConfigPath, nil
}

func ParseConfig() {
	path, err := getConfigPath()
	if err != nil {
		return
	}

	config, err := read(path)
	if err != nil {
		log.Fatal(err)
	}

	if config.Offset != 0 {
		Offset = config.Offset
	}

	if config.Limit != 0 {
		Limit = config.Limit
	}

	if config.Name != "" {
		Name = config.Name
	}

	if config.RawOutput {
		RawOutput = config.RawOutput
	}
}
