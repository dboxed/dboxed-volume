package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/dboxed/dboxed-volume/pkg/server/models"
	"golang.org/x/oauth2"
)

type ClientAuth struct {
	ApiUrl   string           `json:"apiUrl"`
	AuthInfo *models.AuthInfo `json:"authInfo"`
	Token    *oauth2.Token    `json:"token"`
}

func GetClientAuthPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(homeDir, ".dboxed-volume", "client-auth.json")
	return p, nil
}

func ReadClientAuth() (*ClientAuth, error) {
	p, err := GetClientAuthPath()
	if err != nil {
		return nil, err
	}

	b, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	var ret ClientAuth
	err = json.Unmarshal(b, &ret)
	if err != nil {
		return nil, err
	}
	return &ret, nil
}

func WriteClientAuth(ca *ClientAuth) error {
	p, err := GetClientAuthPath()
	if err != nil {
		return err
	}
	err = os.MkdirAll(filepath.Dir(p), 0700)
	if err != nil {
		return err
	}
	b, err := json.Marshal(ca)
	if err != nil {
		return err
	}

	err = os.WriteFile(p, b, 0700)
	if err != nil {
		return err
	}
	return nil
}
