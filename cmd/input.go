package cmd

import "strings"

type Input struct {
	Manager *CommandManager
	Command string
	Args    []string
	Lines   []string
}

func ParseInput(s string, c *CommandManager) *Input {
	if s[0] != '!' {
		return nil
	}

	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return nil
	}

	name, args := ParseCommand(lines[0])
	if name == "" {
		return nil
	}

	if len(lines) > 1 {
		lines = lines[1:]
	}

	return &Input{
		Manager: c,
		Command: name,
		Args:    args,
		Lines:   lines,
	}
}

func ParseCommand(s string) (name string, args []string) {
	if s[0] != '!' {
		return "", nil
	}

	parts := strings.Split(s, " ")

	// name is the first word, minus the '!' prefix
	name = parts[0][1:]
	if len(parts) > 1 {
		// args are the remaining parts of the input
		args = parts[1:]
	}

	return strings.ToLower(name), args
}
