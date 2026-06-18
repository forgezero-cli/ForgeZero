package shell

import (
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/c-bata/go-prompt"
)

var promptNew = func(executor func(string), completer func(prompt.Document) []prompt.Suggest, opts ...prompt.Option) interface{ Run() } {
	return prompt.New(executor, completer, opts...)
}

func Run() {
	state := DefaultState()

	status := "SEALED"
	if os.Getenv("FZ_STAGING") == "1" {
		status = "STAGING"
	}
	os.Stdout.WriteString("FORGEZERO 4.0 ZERO [MIL-SPEC] // STATUS: " + status + " // AUTONOMY: ACTIVE\n")
	os.Stdout.WriteString("┌─────────────────────────────────┐\n")
	os.Stdout.WriteString("│  fz interactive shell           │\n")
	os.Stdout.WriteString("│  Type 'help' for commands       │\n")
	os.Stdout.WriteString("└─────────────────────────────────┘\n")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sc
		os.Stdout.WriteString("\nExiting...\n")
		os.Exit(0)
	}()

	executor := func(s string) {
		if s == "" {
			return
		}
		args := splitCommand(s)
		cmd := args[0]
		switch cmd {
		case "build":
			if err := cmdBuild(state); err != nil {
				os.Stderr.WriteString("error: " + err.Error() + "\n")
			}
		case "clean":
			if err := cmdClean(state); err != nil {
				os.Stderr.WriteString("error: " + err.Error() + "\n")
			}
		case "set":
			if err := cmdSet(state, args); err != nil {
				os.Stderr.WriteString("error: " + err.Error() + "\n")
			}
		case "show":
			cmdShow(state)
		case "watch":
			os.Stdout.WriteString("watch mode coming soon\n")
		case "exit", "quit":
			os.Stdout.WriteString("Goodbye.\n")
			os.Exit(0)
		case "help":
			cmdHelp()
		default:
			os.Stderr.WriteString("unknown command: " + cmd + "\n")
		}
	}

	p := promptNew(
		executor,
		Completer,
		prompt.OptionTitle("fz shell"),
		prompt.OptionPrefix("[fz] > "),
		prompt.OptionPrefixTextColor(prompt.Green),
		prompt.OptionSuggestionBGColor(prompt.DarkGray),
		prompt.OptionSelectedSuggestionBGColor(prompt.Blue),
	)
	p.Run()
}

func splitCommand(s string) []string {
	var parts []string
	var b strings.Builder
	inQuote := false
	for _, ch := range s {
		if ch == '"' {
			inQuote = !inQuote
		} else if ch == ' ' && !inQuote {
			if b.Len() > 0 {
				parts = append(parts, b.String())
				b.Reset()
			}
		} else {
			b.WriteRune(ch)
		}
	}
	if b.Len() > 0 {
		parts = append(parts, b.String())
	}
	return parts
}