package config

import (
	"fmt"
	"os"

	"sigs.k8s.io/yaml"
)

type Config struct {
	Auth   AuthConfig   `json:"auth"`
	DB     DbConfig     `json:"db"`
	Server ServerConfig `json:"server"`
}

type AuthConfig struct {
	OidcIssuerUrl string `json:"oidcIssuerUrl"`
	OidcClientId  string `json:"oidcClientId"`

	AdminUsers []string `json:"adminUsers"`
}

type DbConfig struct {
	Url     string `json:"url"`
	Migrate bool   `json:"migrate"`
}

type ServerConfig struct {
	ListenAddress string `json:"listenAddress"`
	BaseUrl       string `json:"baseUrl"`
}

func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		return nil, fmt.Errorf("missing config path")
	}

	f, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(f, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
