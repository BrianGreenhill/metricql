package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/sashabaranov/go-openai"

	"github.com/briangreenhill/metricql/datadog"
	"github.com/briangreenhill/metricql/mcp"
)

type MetricQuery struct {
	MetricName  string            `json:"MetricName"`
	Aggregation string            `json:"Aggregation"`
	Filters     map[string]string `json:"Filters"`
	TimeWindow  string            `json:"TimeWindow"`
}

const systemPrompt = `
You are an observability expert specializing in Datadog metrics. You are acting as an assistant to help users generate metric queries based on natural language prompts.

Based on the provided system context, translate the user's question into a structured JSON object described in the metric query.

Use this format:
{
	"MetricName": "string",
	"Aggregation": "string",
	"Filters": { "tag_key": "value" },
	"TimeWindow": "1h"
}

Only use valid metrics, aggregations, and filters based on the CONTEXT.
Do not invent fields. If you are unsure, return nulls.
`

func main() {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "metricql > ",
		HistoryFile:     "/tmp/metricql.repl.history",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := rl.Close(); err != nil {
			fmt.Printf("Failed to close readline: %v\n", err)
		}
	}()

	fmt.Println("metricql REPL mode -- type 'help' or 'exit' to quit")

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			break
		}
		switch line {
		case "exit", "quit":
			return
		case "help", "?":
			printHelp()
			continue
		case "clear":
			clearScreen()
			continue
		default:
			runPrompt(line)
		}
	}
}

func executeQuery(ctx context.Context, mq MetricQuery) {
	from, to, err := ParseTimeRange(mq.TimeWindow)
	if err != nil {
		fmt.Printf("Error parsing time range: %v\n", err)
		return
	}

	query := buildQueryString(mq)

	client := datadog.NewClient()
	summary, err := client.QueryMetrics(ctx, query, from, to, "ms")
	if err != nil {
		fmt.Printf("Error querying Datadog: %v\n", err)
		return
	}
	fmt.Println("Query Summary:", summary)
}

func buildQueryString(mq MetricQuery) string {
	tags := ""
	if len(mq.Filters) > 0 {
		tagParts := []string{}
		for k, v := range mq.Filters {
			tagParts = append(tagParts, fmt.Sprintf("%s:%s", k, v))
		}
		tags = "{" + strings.Join(tagParts, ",") + "}"
	}
	return fmt.Sprintf("%s:%s%s", mq.Aggregation, mq.MetricName, tags)
}

func ParseTimeRange(durStr string) (time.Time, time.Time, error) {
	dur, err := time.ParseDuration(durStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid duration format: %w", err)
	}
	to := time.Now()
	from := to.Add(-dur)
	return from, to, nil
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func printHelp() {
	fmt.Println(`
🆘 metricql REPL Help

Type natural language prompts to query Datadog metrics.

Examples:
  99th percentile latency for unicorn over the last 15 minutes
  avg latency for unicorn
  max error rate for unicorn-api in prod last hour

Commands:
  help       Show this help message
  exit       Exit the REPL

Tip:
  Use metric names like "latency", "errors", or "rps"
  Use time phrases like "last hour", "past 30 minutes"
  Use filters like "for unicorn", "in prod", "region us-west"
		`)
}

func runPrompt(promptText string) {
	mcpCtx, err := mcp.LoadMCPContext("ontology.yaml")
	if err != nil {
		fmt.Printf("Failed to load MCP context: %v\n", err)
		return
	}

	jsonBytes, err := mcp.BuildLLMContext(mcpCtx)
	if err != nil {
		fmt.Printf("Failed to build LLM context: %v\n", err)
		return
	}

	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt + "\n\nCONTEXT:\n" + string(jsonBytes),
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: promptText,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       openai.GPT4Dot1,
		Messages:    messages,
		Temperature: 0.2,
	})
	if err != nil {
		fmt.Printf("Error calling OpenAI API: %v\n", err)
		return
	}
	output := resp.Choices[0].Message.Content
	// debug output to save API tokens
	// 		output := `
	// {
	//         "MetricName": "request.dist.time",
	//         "Aggregation": "p99",
	//         "Filters": { "kube_deployment": "unicorn-api" },
	//         "TimeWindow": "24h"
	// }
	// `
	fmt.Println("LLM Response:", output)

	var query MetricQuery
	if err := json.Unmarshal([]byte(output), &query); err != nil {
		fmt.Printf("Error parsing LLM response: %v\n", err)
	}

	executeQuery(ctx, query)
}
