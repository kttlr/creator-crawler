package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"creator-crawler/internal/config"
	"creator-crawler/internal/export"
	"creator-crawler/internal/youtube"
)

const usage = `Usage:
  creator-crawler search "Game Name" [--limit 100] [--output-dir outputs] [--include-filtered]

Commands:
  search    Search YouTube for creator videos about a game

Flags:
  --limit int          Number of videos to fetch before filtering (default 100)
  --output-dir string  Directory for timestamped CSV files (default outputs)
  --include-filtered   Include likely official/media/trailer rows with filtered_reason
  --help               Show help
`

type searchOptions struct {
	Query           string
	Limit           int
	OutputDir       string
	IncludeFiltered bool
}

func Run(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Print(usage)
		return nil
	}

	switch args[0] {
	case "search":
		return runSearch(ctx, args[1:])
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], usage)
	}
}

func runSearch(ctx context.Context, args []string) error {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Print(usage)
		return nil
	}

	options, err := parseSearchOptions(args)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := youtube.NewClient(cfg.YouTubeAPIKey)
	results, err := client.Search(ctx, options.Query, options.Limit, options.IncludeFiltered)
	if err != nil {
		return err
	}

	path, err := export.WriteCSV(results, options.Query, options.OutputDir)
	if err != nil {
		return err
	}

	filteredNote := ""
	if !options.IncludeFiltered {
		filteredNote = " likely official/media/trailer results excluded"
	}
	fmt.Printf("Wrote %d videos to %s.%s\n", len(results), path, filteredNote)
	return nil
}

func parseSearchOptions(args []string) (searchOptions, error) {
	options := searchOptions{Limit: 100, OutputDir: "outputs"}
	var queryParts []string
	if len(args) == 0 {
		return options, errors.New("missing game name\n\n" + usage)
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--help" || arg == "-h":
			fmt.Print(usage)
			return options, nil
		case arg == "--include-filtered":
			options.IncludeFiltered = true
		case arg == "--limit":
			if i+1 >= len(args) {
				return options, errors.New("--limit requires a value")
			}
			limit, err := strconv.Atoi(args[i+1])
			if err != nil || limit <= 0 {
				return options, errors.New("--limit must be a positive integer")
			}
			options.Limit = limit
			i++
		case strings.HasPrefix(arg, "--limit="):
			limit, err := strconv.Atoi(strings.TrimPrefix(arg, "--limit="))
			if err != nil || limit <= 0 {
				return options, errors.New("--limit must be a positive integer")
			}
			options.Limit = limit
		case arg == "--output-dir":
			if i+1 >= len(args) {
				return options, errors.New("--output-dir requires a value")
			}
			options.OutputDir = args[i+1]
			i++
		case strings.HasPrefix(arg, "--output-dir="):
			options.OutputDir = strings.TrimPrefix(arg, "--output-dir=")
		case strings.HasPrefix(arg, "--"):
			return options, fmt.Errorf("unknown flag %q", arg)
		default:
			queryParts = append(queryParts, arg)
		}
	}

	options.Query = strings.TrimSpace(strings.Join(queryParts, " "))
	if options.Query == "" {
		return options, errors.New("missing game name\n\n" + usage)
	}
	if strings.TrimSpace(options.OutputDir) == "" {
		return options, errors.New("--output-dir cannot be blank")
	}

	return options, nil
}
