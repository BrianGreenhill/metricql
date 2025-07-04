package mcp

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type MCPContext struct {
	Services map[string]Service `yaml:"services"`
	Metrics  map[string]Metric  `yaml:"metrics"`
	Teams    map[string]Team    `yaml:"teams"`
	Aliases  map[string]string  `yaml:"aliases"`
}

type Service struct {
	Description string            `yaml:"description"`
	Tags        map[string]string `yaml:"tags"`
	Team        string            `yaml:"team"`
	Metrics     []string          `yaml:"metrics"`
}

type Metric struct {
	Description string          `yaml:"description"`
	Type        string          `yaml:"type"` // e.g., "gauge", "counter"
	Tags        []string        `yaml:"tags"`
	Supports    map[string]View `yaml:"supports"` // e.g., "p99", "avg"
}

type View struct {
	Type         string   `yaml:"type"`                    // e.g., "percentile", "average"
	Percentiles  []string `yaml:"percentiles,omitempty"`   // e.g., ["p95", "p99"]
	Aggregation  string   `yaml:"aggregation"`             // e.g., "avg", "sum"
	Unit         string   `yaml:"unit,omitempty"`          // e.g., "ms", "requests/s"
	Filter       string   `yaml:"filter,omitempty"`        // e.g., "service:my-service"
	ExampleQuery string   `yaml:"example_query,omitempty"` // Example query for this view
}

type Team struct {
	OnCall   string   `yaml:"on_call"`  // Email or contact for on-call
	Services []string `yaml:"services"` // List of services owned by this team
}

func LoadMCPContext(path string) (*MCPContext, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP context file: %w", err)
	}
	var ctx MCPContext
	if err := yaml.Unmarshal(data, &ctx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal MCP context: %w", err)
	}

	return &ctx, nil
}
