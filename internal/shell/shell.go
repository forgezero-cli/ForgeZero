package shell

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/c-bata/go-prompt"
)

func Run() {
	state := DefaultState()

	fmt.Println("┌─────────────────────────────────┐")
	fmt.Println("│  fz interactive shell           │")
	fmt.Println("│  Type 'help' for commands       │")
	fmt.Println("└─────────────────────────────────┘")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sc
		fmt.Println("\nExiting...")
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
				fmt.Printf("error: %v\n", err)
			}
		case "clean":
			if err := cmdClean(state); err != nil {
				fmt.Printf("error: %v\n", err)
			}
		case "set":
			if err := cmdSet(state, args); err != nil {
				fmt.Printf("error: %v\n", err)
			}
		case "show":
			cmdShow(state)
		case "watch":
			fmt.Println("watch mode coming soon")
		case "exit", "quit":
			fmt.Println("Goodbye.")
			os.Exit(0)
		case "help":
			cmdHelp()
		default:
			fmt.Printf("unknown command: %s\n", cmd)
		}
	}

	p := prompt.New(
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
	var res []string
	var cur string
	inQuote := false
	for _, ch := range s {
		if ch == '"' {
			inQuote = !inQuote
		} else if ch == ' ' && !inQuote {
			if cur != "" {
				res = append(res, cur)
				cur = ""
			}
		} else {
			cur += string(ch)
		}
	}
	if cur != "" {
		res = append(res, cur)
	}
	return res
}
