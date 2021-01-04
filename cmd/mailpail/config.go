package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/hclsimple"
)

type Config struct {
	API     *ConfigAPI `hcl:"api,block"`
	Maildir string     `hcl:"maildir"`
}

type ConfigAPI struct {
	Endpoint  string `hcl:"endpoint"`
	Token     string `hcl:"token,optional"`
	TokenFile string `hcl:"tokenFile,optional"`
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

	files := []string{"mailpail.hcl", "mailpail.json"}

	var conf Config
	for _, file := range files {
		name := filepath.Join(configDir, file)
		fi, err := os.Stat(name)

		switch {
		case os.IsNotExist(err):
			continue
		case err != nil:
			return Config{}, err
		case !fi.Mode().IsRegular():
			return Config{}, fmt.Errorf("file is not a regular file: %s", name)
		}

		if err := hclsimple.DecodeFile(filepath.Join(configDir, file), nil, &conf); err != nil {
			continue
		}

		return conf, nil
	}

	return Config{}, fmt.Errorf("unable to load configuration files")
}
