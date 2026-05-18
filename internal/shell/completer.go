package shell

import (
	"strings"

	"github.com/c-bata/go-prompt"
)

var suggestions = []prompt.Suggest{
	{Text: "build", Description: "Build project"},
	{Text: "clean", Description: "Clean artifacts"},
	{Text: "set", Description: "Set configuration key=value"},
	{Text: "show", Description: "Show current settings"},
	{Text: "watch", Description: "Start watch mode"},
	{Text: "exit", Description: "Exit shell"},
	{Text: "quit", Description: "Exit shell"},
	{Text: "help", Description: "Show help"},
}

func Completer(in prompt.Document) []prompt.Suggest {
	w := in.GetWordBeforeCursor()
	if w == "" {
		return []prompt.Suggest{}
	}
	if strings.HasPrefix(w, "set ") {
		parts := strings.SplitN(in.Text, " ", 3)
		if len(parts) >= 2 && parts[1] == "set" && len(parts) < 3 {
			return prompt.FilterHasPrefix([]prompt.Suggest{
				{Text: "mode="},
				{Text: "format="},
				{Text: "strict="},
				{Text: "sanitize="},
				{Text: "verbose="},
				{Text: "debug="},
				{Text: "no-cache="},
				{Text: "no-symbol-check="},
				{Text: "keep-obj="},
				{Text: "ld-script="},
				{Text: "text-addr="},
				{Text: "out="},
			}, in.GetWordBeforeCursor(), true)
		}
		return prompt.FilterHasPrefix(suggestions, w, true)
	}
	return prompt.FilterHasPrefix(suggestions, w, true)
}
