package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"creator-crawler/internal/config"
	"creator-crawler/internal/model"
	"creator-crawler/internal/server"
	"creator-crawler/internal/storage"
	"creator-crawler/internal/youtube"
)

const usage = `Usage:
  creator-crawler game add "Game Name" [--db data/creator-crawler.db]
  creator-crawler game list [--db data/creator-crawler.db]
  creator-crawler search --game "Game Name" [--query "Search Query"] [--limit 100] [--db data/creator-crawler.db] [--include-filtered]
  creator-crawler serve [--addr localhost:8080] [--db data/creator-crawler.db]

Commands:
  game add    Add a tracked game
  game list   List tracked games
  search      Search YouTube for videos tied to an existing game
  serve       Start the local web UI

Flags:
  --game string       Existing game to tie search results to
  --query string      YouTube search query (defaults to game name)
  --limit int         Number of videos to fetch before filtering (default 100)
  --db string         SQLite database path (default data/creator-crawler.db)
  --addr string       HTTP address for serve (default localhost:8080)
  --include-filtered  Include likely official/media/trailer rows with filtered_reason
  --help              Show help
`

type gameOptions struct {
	Name string
	DB   string
}

type searchOptions struct {
	GameName        string
	Query           string
	Limit           int
	DB              string
	IncludeFiltered bool
}

type serveOptions struct {
	Addr string
	DB   string
}

func Run(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		fmt.Print(usage)
		return nil
	}

	switch args[0] {
	case "game":
		return runGame(ctx, args[1:])
	case "search":
		return runSearch(ctx, args[1:])
	case "serve":
		return runServe(ctx, args[1:])
	default:
		return fmt.Errorf("unknown command %q\n\n%s", args[0], usage)
	}
}

func runServe(ctx context.Context, args []string) error {
	if hasHelp(args) {
		fmt.Print(usage)
		return nil
	}
	options, err := parseServeOptions(args)
	if err != nil {
		return err
	}
	return server.Serve(ctx, options.Addr, options.DB)
}

func runGame(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("missing game command: use 'game add' or 'game list'\n\n" + usage)
	}
	if hasHelp(args) {
		fmt.Print(usage)
		return nil
	}

	switch args[0] {
	case "add":
		options, err := parseGameOptions(args[1:], true)
		if err != nil {
			return err
		}

		store, err := storage.Open(options.DB)
		if err != nil {
			return err
		}
		defer store.Close()

		game, err := store.AddGame(ctx, options.Name)
		if err != nil {
			return err
		}

		fmt.Printf("Added game %q with id %d.\n", game.Name, game.ID)
		return nil
	case "list":
		options, err := parseGameOptions(args[1:], false)
		if err != nil {
			return err
		}

		store, err := storage.Open(options.DB)
		if err != nil {
			return err
		}
		defer store.Close()

		games, err := store.ListGames(ctx)
		if err != nil {
			return err
		}
		if len(games) == 0 {
			fmt.Println("No games tracked yet. Add one with: creator-crawler game add \"Game Name\"")
			return nil
		}

		for _, game := range games {
			fmt.Printf("%d\t%s\t%d videos\n", game.ID, game.Name, game.VideoCount)
		}
		return nil
	default:
		return fmt.Errorf("unknown game command %q\n\n%s", args[0], usage)
	}
}

func runSearch(ctx context.Context, args []string) error {
	if hasHelp(args) {
		fmt.Print(usage)
		return nil
	}

	options, err := parseSearchOptions(args)
	if err != nil {
		return err
	}

	store, err := storage.Open(options.DB)
	if err != nil {
		return err
	}
	defer store.Close()

	game, found, err := store.FindGameByName(ctx, options.GameName)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("game %q does not exist.\nRun: creator-crawler game add %q", options.GameName, options.GameName)
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

	searchRunID, err := store.SaveSearchResults(ctx, game, options.Query, options.Limit, options.IncludeFiltered, results)
	if err != nil {
		return err
	}

	filteredNote := ""
	if !options.IncludeFiltered {
		filteredNote = " Likely official/media/trailer results excluded."
	}
	fmt.Printf("Stored search run %d for %q.\n", searchRunID, game.Name)
	fmt.Printf("Saved %d videos to %s.%s\n", len(results), options.DB, filteredNote)
	printTopVideos(results, 10)
	return nil
}

func parseGameOptions(args []string, requireName bool) (gameOptions, error) {
	options := gameOptions{DB: storage.DefaultDBPath}
	var nameParts []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--help" || arg == "-h":
			fmt.Print(usage)
			return options, nil
		case arg == "--db":
			if i+1 >= len(args) {
				return options, errors.New("--db requires a value")
			}
			options.DB = args[i+1]
			i++
		case strings.HasPrefix(arg, "--db="):
			options.DB = strings.TrimPrefix(arg, "--db=")
		case strings.HasPrefix(arg, "--"):
			return options, fmt.Errorf("unknown flag %q", arg)
		default:
			nameParts = append(nameParts, arg)
		}
	}

	options.Name = strings.TrimSpace(strings.Join(nameParts, " "))
	if requireName && options.Name == "" {
		return options, errors.New("missing game name\n\n" + usage)
	}
	if strings.TrimSpace(options.DB) == "" {
		return options, errors.New("--db cannot be blank")
	}

	return options, nil
}

func parseSearchOptions(args []string) (searchOptions, error) {
	options := searchOptions{Limit: 100, DB: storage.DefaultDBPath}
	if len(args) == 0 {
		return options, errors.New("missing --game\n\n" + usage)
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--help" || arg == "-h":
			fmt.Print(usage)
			return options, nil
		case arg == "--include-filtered":
			options.IncludeFiltered = true
		case arg == "--game":
			value, next, err := readFlagValue(args, i, "--game")
			if err != nil {
				return options, err
			}
			options.GameName = value
			i = next
		case strings.HasPrefix(arg, "--game="):
			options.GameName = strings.TrimPrefix(arg, "--game=")
		case arg == "--query":
			value, next, err := readFlagValue(args, i, "--query")
			if err != nil {
				return options, err
			}
			options.Query = value
			i = next
		case strings.HasPrefix(arg, "--query="):
			options.Query = strings.TrimPrefix(arg, "--query=")
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
		case arg == "--db":
			value, next, err := readFlagValue(args, i, "--db")
			if err != nil {
				return options, err
			}
			options.DB = value
			i = next
		case strings.HasPrefix(arg, "--db="):
			options.DB = strings.TrimPrefix(arg, "--db=")
		case strings.HasPrefix(arg, "--"):
			return options, fmt.Errorf("unknown flag %q", arg)
		default:
			return options, fmt.Errorf("unexpected argument %q; use --game and --query", arg)
		}
	}

	options.GameName = strings.TrimSpace(options.GameName)
	if options.GameName == "" {
		return options, errors.New("missing --game\n\n" + usage)
	}

	options.Query = strings.TrimSpace(options.Query)
	if options.Query == "" {
		options.Query = options.GameName
	}
	if strings.TrimSpace(options.DB) == "" {
		return options, errors.New("--db cannot be blank")
	}

	return options, nil
}

func readFlagValue(args []string, index int, name string) (string, int, error) {
	if index+1 >= len(args) {
		return "", index, fmt.Errorf("%s requires a value", name)
	}
	return args[index+1], index + 1, nil
}

func parseServeOptions(args []string) (serveOptions, error) {
	options := serveOptions{Addr: "localhost:8080", DB: storage.DefaultDBPath}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--addr":
			value, next, err := readFlagValue(args, i, "--addr")
			if err != nil {
				return options, err
			}
			options.Addr = value
			i = next
		case strings.HasPrefix(arg, "--addr="):
			options.Addr = strings.TrimPrefix(arg, "--addr=")
		case arg == "--db":
			value, next, err := readFlagValue(args, i, "--db")
			if err != nil {
				return options, err
			}
			options.DB = value
			i = next
		case strings.HasPrefix(arg, "--db="):
			options.DB = strings.TrimPrefix(arg, "--db=")
		case strings.HasPrefix(arg, "--"):
			return options, fmt.Errorf("unknown flag %q", arg)
		default:
			return options, fmt.Errorf("unexpected argument %q", arg)
		}
	}
	if strings.TrimSpace(options.Addr) == "" {
		return options, errors.New("--addr cannot be blank")
	}
	if strings.TrimSpace(options.DB) == "" {
		return options, errors.New("--db cannot be blank")
	}
	return options, nil
}

func hasHelp(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

func printTopVideos(results []model.Result, limit int) {
	if len(results) == 0 {
		return
	}
	if len(results) < limit {
		limit = len(results)
	}

	fmt.Println("\nTop videos:")
	for i := 0; i < limit; i++ {
		result := results[i]
		fmt.Printf("%d. %s | %d views | %s\n", i+1, result.CreatorName, result.ViewCount, result.VideoTitle)
	}
}
