package cli

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/pelletier/go-toml"
)

type Config struct {
	Offset    uint   `toml:"offset"`
	Limit     uint   `toml:"limit"`
	Name      string `toml:"name"`
	RawOutput bool   `toml:"raw_output"`
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

func ParseConfig() {
	if ConfigPath == "" {
		// No config file
		return
	}

	if _, err := os.Stat(ConfigPath); os.IsNotExist(err) {
		errConfigNotFound := errors.Wrap(errors.New("config file was not found"), err)
		logError(errConfigNotFound)
		return
	}

	config, err := read(ConfigPath)
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
