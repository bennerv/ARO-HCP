// Copyright 2025 Microsoft Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// QuotaRequest represents a single quota request for a resource
type QuotaRequest struct {
	Provider     string `yaml:"provider"`
	ResourceName string `yaml:"resourceName"`
	Limit        int    `yaml:"limit"`
}

// Config represents the configuration for quota requests
type Config struct {
	Quotas []QuotaRequest `yaml:"quotas"`
}

// LoadConfig loads the quota configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if len(c.Quotas) == 0 {
		return fmt.Errorf("no quota requests defined")
	}

	for i, quota := range c.Quotas {
		if quota.Provider == "" {
			return fmt.Errorf("quota[%d]: provider is required", i)
		}
		if quota.ResourceName == "" {
			return fmt.Errorf("quota[%d]: resourceName is required", i)
		}
		if quota.Limit <= 0 {
			return fmt.Errorf("quota[%d]: limit must be positive", i)
		}
	}

	return nil
}
