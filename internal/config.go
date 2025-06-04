package internal

import (
	"os"

	"github.com/Cadeusept/dependency-validator/internal/entities"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Repos []entities.Repo `yaml:"repos"`
}

func LoadConfig(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	err = yaml.Unmarshal(data, &cfg)
	return cfg, err
}
