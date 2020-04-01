package cmd

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
)

type CommandManager struct {
	path     string
	mutex    *sync.RWMutex
	commands map[string]*Command
}

func NewManager(path string) *CommandManager {
	return &CommandManager{
		path:     path,
		mutex:    &sync.RWMutex{},
		commands: map[string]*Command{},
	}
}

func (c *CommandManager) Register(name string, cmd *Command) string {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	existing, ok := c.commands[name]
	if ok && existing.Fixed {
		return "Cannot replace that command"
	}
	c.commands[name] = cmd
	return "Registered command `!" + name + "`"
}

func (c *CommandManager) RegisterExec(name string, perms []string, exec Executor) string {
	return c.Register(name, &Command{
		Fixed: true,
		Perms: perms,
		Exec:  exec,
	})
}

func (c *CommandManager) Unregister(name string) string {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	cmd, ok := c.commands[name]
	if !ok {
		return "Executor does not exist"
	}
	if cmd.Fixed {
		return "Executor cannot be unregistered"
	}
	delete(c.commands, name)
	return "Executor unregistered"
}

func (c *CommandManager) List(subject Subject) []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	var names []string
	for k, c := range c.commands {
		if c.Test(subject) {
			names = append(names, k)
		}
	}
	sort.Strings(names)
	return names
}

func (c *CommandManager) Each(consumer func(k string, v *Command)) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	for k, v := range c.commands {
		consumer(k, v)
	}
}

func (c *CommandManager) Process(subject Subject, raw string) (bool, string) {
	input := ParseInput(raw, c)
	if input == nil {
		return false, "Input is not a command"
	}

	c.mutex.RLock()
	command, ok := c.commands[input.CommandRaw]
	if !ok {
		command, ok = c.commands[input.Command]
	}
	c.mutex.RUnlock()

	if !ok {
		return false, "Executor not registered"
	}

	if !command.Test(subject) {
		return false, "No permission"
	}

	return true, command.Exec.Call(subject, input)
}

func (c *CommandManager) Load() {
	f, e := os.Open(c.path)
	if e != nil {
		log.Println(e)
		return
	}
	defer logClose(f)

	var content map[string][]string
	e = json.NewDecoder(f).Decode(&content)
	if e != nil {
		log.Println(e)
		return
	}

	for k, v := range content {
		message := strings.Join(v, "\n")
		c.Register(k, &Command{
			Exec:  &Message{Message: message},
			Fixed: false,
		})
	}

	log.Println("Loaded message commands")
}

func (c *CommandManager) Save() {
	f, e := os.Create(c.path)
	if e != nil {
		log.Println(e)
		return
	}
	defer logClose(f)

	content := map[string][]string{}
	c.Each(func(k string, v *Command) {
		switch t := v.Exec.(type) {
		case *Message:
			content[k] = strings.Split(t.Message, "\n")
		}
	})

	en := json.NewEncoder(f)
	en.SetIndent("", "  ")
	e = en.Encode(content)
	if e != nil {
		log.Println(e)
		return
	}

	log.Println("Saved message commands")
}

func logClose(c io.Closer) {
	if c != nil {
		e := c.Close()
		if e != nil {
			log.Println(e)
		}
	}
}
