package main

import (
	"bytes"
	"context"
	"flag"
	"log"
	"strings"

	"github.com/andersfylling/disgord"

	"github.com/TerraForged/halp/cmd"
)

var (
	ownerId = "99824915045191680"
	token   = flag.String("token", "", "Discord token")
)

func main() {
	flag.Parse()

	commands := cmd.NewManager("commands.json")
	commands.Load()
	setup(commands)
	defer commands.Save()

	bot, e := disgord.NewClient(disgord.Config{BotToken: *token})
	if e != nil {
		panic(e)
	}

	handle(bot, commands)

	e = bot.StayConnectedUntilInterrupted(context.Background())
	if e != nil {
		panic(e)
	}
}

func setup(commands *cmd.CommandManager) {
	commands.Register("help", &cmd.Command{
		Exec:  cmd.Wrap(help),
		Fixed: true,
	})
	commands.Register("learn", &cmd.Command{
		Exec:  cmd.Wrap(learn),
		Fixed: true,
		Perms: []string{ownerId},
	})
	commands.Register("forget", &cmd.Command{
		Exec:  cmd.Wrap(forget),
		Fixed: true,
		Perms: []string{ownerId},
	})
}

func handle(bot *disgord.Client, commands *cmd.CommandManager) {
	bot.On(disgord.EvtGuildCreate, func(s disgord.Session, g *disgord.GuildCreate) {
		log.Println("Joined guild:", g.Guild.Name)
	})

	bot.On(disgord.EvtMessageCreate, func(s disgord.Session, m *disgord.MessageCreate) {
		if m.Message.Author.Bot {
			return
		}

		subject := cmd.NewSubject(m.Message.Author.ID.String())
		if success, message := commands.Process(subject, m.Message.Content); success {
			_, e := s.SendMsg(m.Ctx, m.Message.ChannelID, message)
			if e != nil {
				log.Println(e)
			}
		}
	})
}

func help(i *cmd.Input) string {
	buf := bytes.Buffer{}
	for _, name := range i.Manager.List() {
		if buf.Len() > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString("`!")
		buf.WriteString(name)
		buf.WriteString("`")
	}
	return buf.String()
}

func learn(i *cmd.Input) string {
	if len(i.Args) == 0 {
		return "No keyword/phrase provided"
	}

	if len(i.Lines) < 1 {
		return "No message lines provided"
	}

	name := i.Args[0]
	message := strings.Join(i.Lines, "\n")
	return i.Manager.Register(name, &cmd.Command{
		Exec:  &cmd.Message{Message: message},
		Fixed: false,
	})
}

func forget(i *cmd.Input) string {
	if len(i.Args) == 0 {
		return "No command provided"
	}
	name := strings.Join(i.Args, " ")
	return i.Manager.Unregister(name)
}
