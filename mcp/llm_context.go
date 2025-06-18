package mcp

import "encoding/json"

type LLMService struct {
	Name    string            `json:"name"`
	Team    string            `json:"team"`
	Tags    map[string]string `json:"tags"`
	Metrics []LLMMetric       `json:"metrics"`
}

type LLMMetric struct {
	Name     string             `json:"name"`
	Type     string             `json:"type"`     // e.g., "gauge", "counter"
	Supports map[string]LLMView `json:"supports"` // e.g., "p99", "avg"
}

type LLMView struct {
	Type         string   `json:"type"`                    // e.g., "percentile", "average"
	Aggregation  string   `json:"aggregation"`             // e.g., "avg", "sum"
	Filter       string   `json:"filter,omitempty"`        // e.g., "service:my-service"
	Percentiles  []string `json:"percentiles,omitempty"`   // e.g., ["p95", "p99"]
	Unit         string   `json:"unit,omitempty"`          // e.g., "ms", "requests/s"
	ExampleQuery string   `json:"example_query,omitempty"` // Example query for this view
}

func BuildLLMContext(ctx *MCPContext) ([]byte, error) {
	var services []LLMService

	for name, svc := range ctx.Services {
		llmSvc := LLMService{
			Name: name,
			Team: svc.Team,
			Tags: svc.Tags,
		}

		for _, metricName := range svc.Metrics {
			baseMetric, ok := ctx.Metrics[metricName]
			if !ok {
				continue
			}

			llmMetric := LLMMetric{
				Name:     metricName,
				Type:     baseMetric.Type,
				Supports: make(map[string]LLMView),
			}

			for viewKey, view := range baseMetric.Supports {
				llmMetric.Supports[viewKey] = LLMView{
					Type:         view.Type,
					Aggregation:  view.Aggregation,
					Filter:       view.Filter,
					Percentiles:  view.Percentiles,
					Unit:         view.Unit,
					ExampleQuery: view.ExampleQuery,
				}
			}

			llmSvc.Metrics = append(llmSvc.Metrics, llmMetric)
		}

		services = append(services, llmSvc)
	}

	return json.MarshalIndent(map[string]any{
		"services": services,
	}, "", "  ")
}
