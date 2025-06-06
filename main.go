package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"

	"github.com/briangreenhill/metricql/datadog"
	"github.com/briangreenhill/metricql/prompt"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: metricql <prompt> or metricql repl")
		os.Exit(1)
	}
	if os.Args[1] == "repl" {
		startREPL()
		return
	}

	runPrompt(strings.Join(os.Args[1:], " "))
}

func startREPL() {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "metricql > ",
		HistoryFile:     "/tmp/metricql.repl.history",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		panic(err)
	}
	defer rl.Close()

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
	q := prompt.ParsePrompt(promptText)
	fmt.Printf("📊 Generated Metric Query: %+v\n", q)

	queryStr := q.ToDatadogQuery()
	fmt.Println("🔍 Datadog Query String:", queryStr)

	from := time.Now().Add(-prompt.ParseDuration(q.TimeWindow))
	to := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := datadog.NewClient()
	res, err := client.QueryMetrics(ctx, queryStr, from, to)
	if err != nil {
		fmt.Println("❌ Error querying Datadog:", err)
		return
	}

	fmt.Println(res)
}
