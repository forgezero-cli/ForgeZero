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

	fmt.Println("fz interactive shell. Type 'help' for commands.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sc
		fmt.Println("\nExiting...")
		os.Exit(0)
	}()

	for {
		input := prompt.Input("fz> ", Completer)
		if input == "" {
			continue
		}
		args := splitCommand(input)
		cmd := args[0]
		switch cmd {
		case "build":
			if err := cmdBuild(state); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "clean":
			if err := cmdClean(state); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "set":
			if err := cmdSet(state, args); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "show":
			cmdShow(state)
		case "watch":
			fmt.Println("Watch mode not yet implemented in shell.")
		case "exit", "quit":
			fmt.Println("Bye!")
			return
		case "help":
			cmdHelp()
		default:
			fmt.Printf("Unknown command: %s\n", cmd)
		}
	}
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
