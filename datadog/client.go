package datadog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Client struct {
	apiKey     string
	appKey     string
	baseURL    string
	httpClient *http.Client
}

type QueryResult struct {
	Series []Series `json:"series"`
}

type Series struct {
	Metric      string      `json:"metric"`
	PointList   [][]float64 `json:"pointlist"`
	Scope       string      `json:"scope"`
	Expression  string      `json:"expression,omitempty"`
	DisplayName string      `json:"display_name,omitempty"`
}

func NewClient() *Client {
	return &Client{
		apiKey:     os.Getenv("DD_API_KEY"),
		appKey:     os.Getenv("DD_APP_KEY"),
		baseURL:    "https://api.datadoghq.com/api/v1/",
		httpClient: &http.Client{},
	}
}

func (c *Client) QueryMetrics(ctx context.Context, query string, from, to time.Time) (string, error) {
	endpoint := fmt.Sprintf("%s/query", c.baseURL)

	params := url.Values{}
	params.Add("from", fmt.Sprintf("%d", from.Unix()))
	params.Add("to", fmt.Sprintf("%d", to.Unix()))
	params.Add("query", query)

	reqURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("DD-API-KEY", c.apiKey)
	req.Header.Set("DD-APPLICATION-KEY", c.appKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("error response from API: %s", body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var result QueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(result.Series) == 0 {
		return "No data found for the given query", nil
	}

	s := result.Series[0]
	if len(s.PointList) == 0 {
		return "No data points found for the given query", nil
	}

	lastPoint := s.PointList[len(s.PointList)-1]
	timestamp := time.Unix(int64(lastPoint[0]), 0)
	value := lastPoint[1]

	summary := fmt.Sprintf(
		"📊 Metric: %s\nQuery: %s\nTimestamp: %s\nValue: %.2f ms",
		s.DisplayName,
		s.Expression,
		timestamp.Format(time.RFC822),
		value,
	)

	return summary, nil
}
