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
	perms = []string{"halp-admin"}
	token = flag.String("token", "", "Discord token")
)

type DiscordSubject struct {
	roles []string
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
	commands.Register("list", &cmd.Command{
		Exec:  cmd.Wrap(list),
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
	bot.On(disgord.EvtReady, func(s disgord.Session, r *disgord.Ready) {
		log.Println("Setting status")
		e := s.UpdateStatusString("!list")
		if e != nil {
			log.Println(e)
		}
	})

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
			roles: nil,
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

func list(s cmd.Subject, i *cmd.Input) string {
	buf := bytes.Buffer{}
	for _, name := range i.Manager.List(s) {
		if buf.Len() > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString("`!")
		buf.WriteString(name)
		buf.WriteString("`")
	}
	return buf.String()
}

func learn(_ cmd.Subject, i *cmd.Input) string {
	if len(i.Args) == 0 {
		return "No keyword/phrase provided"
	}

	if len(i.Lines) < 1 {
		return "No message lines provided"
	}

	defer i.Manager.Save()

	name := strings.Join(i.Args, " ")
	message := strings.Join(i.Lines, "\n")
	return i.Manager.Register(name, &cmd.Command{
		Exec:  &cmd.Message{Message: message},
		Fixed: false,
	})
}

func forget(_ cmd.Subject, i *cmd.Input) string {
	if len(i.Args) == 0 {
		return "No command provided"
	}
	name := strings.Join(i.Args, " ")
	return i.Manager.Unregister(name)
}

func (s *DiscordSubject) Perms() []string {
	if s.roles != nil {
		return s.roles
	}

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

	s.roles = make([]string, len(m.Roles))
	for i, rid := range m.Roles {
		name := name(roles, rid)
		if name == "" {
			return nil
		}
		s.roles[i] = name
	}
	return s.roles
}

func name(array []*disgord.Role, value disgord.Snowflake) string {
	for _, v := range array {
		if v.ID == value {
			return v.Name
		}
	}
	return ""
}
