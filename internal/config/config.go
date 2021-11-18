package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

const (
	ExpireNever = -1

	EnvDefaultFileStoragePath = "/var/lib/aqua/files/"
	EnvDefaultMetaDbPath      = "/var/lib/aqua/"
)

type AuthConfig struct {
	ValidTokens []*TokenConfig `yaml:"validTokens"`
}

type TokenConfig struct {
	Token string

	// All file types that one can upload via this token.
	// If empty, all file types are allowed.
	ValidFileTypes []string `yaml:"fileTypes"`
}

func NewEmptyAuthConfig() *AuthConfig {
	return &AuthConfig{
		ValidTokens: []*TokenConfig{},
	}
}

func FromData(data []byte) (*AuthConfig, error) {
	var ac AuthConfig
	err := yaml.Unmarshal(data, &ac)
	if err != nil {
		return nil, err
	}
	return &ac, nil
}

func FromLocalFile(path string) (*AuthConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return FromData(data)
}

func (ac *AuthConfig) HasToken(token string) bool {
	for _, validToken := range ac.ValidTokens {
		if validToken.Token == token {
			return true
		}
	}
	return false
}

func (ac *AuthConfig) CanUpload(token string, filetype string) bool {
	for _, validToken := range ac.ValidTokens {
		if validToken.Token == token {
			ft := validToken.ValidFileTypes
			if len(ft) == 0 {
				return true
			}

			for _, s := range ft {
				if s == filetype {
					return true
				}
			}
		}
	}
	return false
}
