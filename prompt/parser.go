package prompt

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type MetricQuery struct {
	MetricName  string
	Aggregation string
	Filters     map[string]string
	TimeWindow  string
}

func ParseDuration(durationStr string) time.Duration {
	d, err := time.ParseDuration(durationStr)
	if err != nil {
		return time.Hour // Default to 1 hour if parsing fails
	}

	return d
}

func (q *MetricQuery) ToDatadogQuery() string {
	filterParts := []string{}
	for k, v := range q.Filters {
		filterParts = append(filterParts, fmt.Sprintf("%s:%s", k, v))
	}
	return fmt.Sprintf("%s:%s{%s}", q.Aggregation, q.MetricName, strings.Join(filterParts, ","))
}

func ParsePrompt(prompt string) MetricQuery {
	prompt = strings.ToLower(prompt)

	aggregation := ""
	switch {
	case strings.Contains(prompt, "p99") || strings.Contains(prompt, "99th"):
		aggregation = "p99"
	case strings.Contains(prompt, "p95") || strings.Contains(prompt, "95th"):
		aggregation = "p95"
	case strings.Contains(prompt, "average") || strings.Contains(prompt, "avg"):
		aggregation = "avg"
	case strings.Contains(prompt, "sum"):
		aggregation = "sum"
	case strings.Contains(prompt, "count"):
		aggregation = "count"
	case strings.Contains(prompt, "max"):
		aggregation = "max"
	case strings.Contains(prompt, "min"):
		aggregation = "min"
	default:
		aggregation = "avg" // Default to average if no specific aggregation is mentioned
	}

	metric := ""
	switch {
	case strings.Contains(prompt, "latency") || strings.Contains(prompt, "response time"):
		metric = "request.dist.time"
	case strings.Contains(prompt, "error rate") || strings.Contains(prompt, "error count") || strings.Contains(prompt, "errors"):
		metric = "request.dist.errors"
	case strings.Contains(prompt, "throughput") || strings.Contains(prompt, "requests per second") || strings.Contains(prompt, "rps"):
		metric = "request.dist.time"
	}

	filters := map[string]string{}
	switch {
	case strings.Contains(prompt, "unicorn"):
		filters["kube_deployment"] = "unicorn"
	}

	timeWindow := extractTimeWindow(prompt)
	if timeWindow == "" {
		timeWindow = "1h" // Default to 1 hour if no time window is specified
	}

	return MetricQuery{
		MetricName:  metric,
		Aggregation: aggregation,
		Filters:     filters,
		TimeWindow:  timeWindow,
	}
}

func extractTimeWindow(prompt string) string {
	patterns := map[string]string{
		`last (\d+) minutes?`: "${1}m",
		`past (\d+) minutes?`: "${1}m",
		`last (\d+) hours?`:   "${1}h",
		`past (\d+) hours?`:   "${1}h",
		`last hour`:           "1h",
		`last day`:            "24h",
		`last minute`:         "1m",
	}

	for pat, replacement := range patterns {
		re := regexp.MustCompile(pat)
		if loc := re.FindStringSubmatchIndex(prompt); loc != nil {
			out := re.ExpandString([]byte{}, replacement, prompt, loc)
			return string(out)
		}
	}

	return ""
}
