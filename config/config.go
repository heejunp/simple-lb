package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 구조체는 config.yaml 파일의 구조와 정확히 일치해야한다.
// 구조체 태그를 통해 매핑한다.
type Config struct {
	Port     string   `yaml:"port"`
	Strategy string   `yaml:"strategy"`
	Backends []string `yaml:"backends"`
}

// Load 함수는 config.yaml 파일을 읽어와 Config 구조체로 반환한다.
func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}