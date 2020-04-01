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
	perms = []string{"dev"}
	token = flag.String("token", "", "Discord token")
)

type DiscordSubject struct {
	ctx   context.Context
	id    disgord.Snowflake
	guild disgord.Snowflake
	sess  disgord.Session
}

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
		Perms: perms,
	})
	commands.Register("forget", &cmd.Command{
		Exec:  cmd.Wrap(forget),
		Fixed: true,
		Perms: perms,
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

		subject := &DiscordSubject{
			sess:  s,
			ctx:   m.Ctx,
			id:    m.Message.Author.ID,
			guild: m.Message.GuildID,
		}

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

func (s *DiscordSubject) Perms() []string {
	m, e := s.sess.GetMember(s.ctx, s.guild, s.id)
	if e != nil {
		log.Println(e)
		return nil
	}

	roles, e := s.sess.GetGuildRoles(s.ctx, s.guild)
	if e != nil {
		log.Println(e)
		return nil
	}

	perms := make([]string, len(m.Roles))
	for i, rid := range m.Roles {
		name := name(roles, rid)
		if name == "" {
			return nil
		}
		perms[i] = name
	}
	return perms
}

func name(array []*disgord.Role, value disgord.Snowflake) string {
	for _, v := range array {
		if v.ID == value {
			return v.Name
		}
	}
	return ""
}
