package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"olympos.io/encoding/edn"
)

type Config struct {
	API      ConfigAPI `edn:"api"`
	Maildir  string    `edn:"maildir"`
	Database string    `edn:"database"`
}

type ConfigAPI struct {
	Endpoint  string `edn:"endpoint"`
	Token     string `edn:"token,omitempty"`
	TokenFile string `edn:"tokenFile,omitempty"`
}

func (c Config) Token() (string, error) {
	switch {
	case len(c.API.TokenFile) > 0:
		b, err := ioutil.ReadFile(c.API.TokenFile)
		if err != nil {
			return "", err
		}

		return string(b), nil
	case len(c.API.Token) > 0:
		return c.API.Token, nil
	}

	return "", fmt.Errorf("api.tokenFile or api.token must be provided")
}

func LoadUserConfig() (Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return Config{}, err
	}
	configDir = filepath.Join(configDir, "mailpail")

	var conf Config
	name := filepath.Join(configDir, "mailpail.edn")
	f, err := os.Open(name)

	switch {
	case os.IsNotExist(err):
		return Config{}, fmt.Errorf("file %q does not exist", name)
	case err != nil:
		return Config{}, fmt.Errorf("unable to open file: %w", err)
	}

	dec := edn.NewDecoder(f)
	if err := dec.Decode(&conf); err != nil {
		return Config{}, fmt.Errorf("unable to parse EDN: %w", err)
	}

	return conf, nil
}
