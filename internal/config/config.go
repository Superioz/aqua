package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

type AuthConfig struct {
	ValidTokens []*TokenConfig `yaml:"validTokens"`
}

type TokenConfig struct {
	Token string
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
