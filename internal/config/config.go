// Package config loads and validates the operator's declared intent from
// YAML: per-table granularity (month/week), pre-provision horizon,
// catch-all size threshold, and retention window. Config lives in files
// under version control, never in the database.
//
// Allowed imports: internal/domain.
package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	DB       []database `yaml:"database"`
	Defaults table      `yaml:"defaults"`
}

type database struct {
	Name     string
	Defaults table `yaml:"defaults"`
	Table    []table
}

type table struct {
	Name        string  `yaml:"name"`
	Granularity *string `yaml:"granularity"`
	Horizon     *int    `yaml:"horizon"`
	Retention   *string `yaml:"retention"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return parse(data)
}

func parse(data []byte) (Config, error) {
	cnf := Config{}
	if err := yaml.Unmarshal([]byte(data), &cnf); err != nil {
		return Config{}, fmt.Errorf("config %w", err)
	}

	return cnf, nil
}

func (cnf *Config) Database(db string) (database, error) {
	for _, v := range cnf.DB {
		if v.Name == db {
			return v, nil
		}
	}
	return database{}, fmt.Errorf("database %q not in config", db)
}
