package cmd

import "strings"

type Input struct {
	Manager    *CommandManager
	Command    string
	CommandRaw string
	Args       []string
	Lines      []string
}

func ParseInput(s string, c *CommandManager) *Input {
	// command must be at least one character in length after '!'
	if len(s) < 2 {
		return nil
	}

	// commands start with '!'
	if s[0] != '!' {
		return nil
	}

	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return nil
	}

	// first line is the command
	raw, name, args := parseCommandParts(lines[0])
	if name == "" {
		return nil
	}

	// drain first line
	if len(lines) == 1 {
		lines = []string{}
	} else {
		lines = lines[1:]
	}

	return &Input{
		Manager:    c,
		Command:    name,
		CommandRaw: raw,
		Args:       args,
		Lines:      lines,
	}
}

func parseCommandParts(s string) (raw, name string, args []string) {
	raw = strings.ToLower(s[1:])
	parts := strings.Split(raw, " ")
	// name is the first word, minus the '!' prefix
	name = parts[0]
	if len(parts) > 1 {
		// args are the remaining parts of the input
		args = parts[1:]
	}
	return raw, name, args
}
