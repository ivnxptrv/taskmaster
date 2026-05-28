package client

import (
	"fmt"
	"github.com/chzyer/readline"
	"log/slog"
	"strings"
	"taskmaster/internal/gateway"
)

// -----------------------------------------------

type Shell struct {
	g   gateway.Gateway
	log *slog.Logger
}

func NewShell(g gateway.Gateway) *Shell {
	return &Shell{g: g}
}

func (s *Shell) Enter() {
	fmt.Println("Shell started")
	// configure the readline instance
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "\033[32mtaskmaster> \033[0m", // Green prompt
		HistoryFile:     "/tmp/taskmaster.history",
		AutoComplete:    nil, // tab completion
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		// panic(err)
		// return nil
	}
	defer rl.Close()

	fmt.Println("Taskmaster Shell. Type 'help' for commands.")

	// loop
	for {
		line, err := rl.Readline()
		if err != nil { // handle Ctrl+D or Ctrl+C
			if err == readline.ErrInterrupt {
				// YOUR CUSTOM LOGIC HERE
				// This is where you call s.g.Shutdown()
				// instead of letting the library handle it
				fmt.Println("\nInterrupt caught by shell loop!")
				s.g.Shutdown()
				break
			}
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// simple parsing
		parts := strings.Fields(line)
		cmd := parts[0]
		args := parts[1:]

		switch cmd {
		case "start":
			if len(args) == 0 {
				fmt.Println("Usage: start <name>")
				continue
			}
			if err := s.g.Start(args[0]); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		case "exit", "quit":
			return
		default:
			fmt.Printf("Unknown command: %s\n", cmd)
		}
	}
}
