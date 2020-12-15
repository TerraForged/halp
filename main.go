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
	perms  = []string{"halp-admin"}
	admins = []string{"dags", "Won-Ton"}
	token  = flag.String("token", "", "Discord token")
)

type DiscordSubject struct {
	roles []string
	ctx   context.Context
	id    disgord.Snowflake
	guild disgord.Snowflake
	sess  disgord.Session
	msg   *disgord.Message
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
	commands.Register("del", &cmd.Command{
		Exec:  cmd.Wrap(del),
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

		if pingBlock(s, m) {
			return
		}

		subject := &DiscordSubject{
			sess:  s,
			msg:   m.Message,
			ctx:   m.Ctx,
			roles: nil,
			id:    m.Message.Author.ID,
			guild: m.Message.GuildID,
		}

		if success, message := commands.Process(subject, m.Message.Content); success && message != "" {
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

func del(s cmd.Subject, i *cmd.Input) string {
	sub, ok := s.(*DiscordSubject)
	if !ok {
		return "Internal error :S"
	}

	params := &disgord.GetMessagesParams{
		After: disgord.ParseSnowflakeString(i.Args[0]),
	}

	// get messages since the 'from' id (arg[0])
	results, e := sub.sess.GetMessages(sub.ctx, sub.msg.ChannelID, params)
	if e != nil {
		return e.Error()
	}

	// get messages up to the 'to' id (arg[1])
	before := ""
	if len(i.Args) == 2 {
		before = i.Args[1]
	}

	// holds a list of message id's to delete
	delParams := &disgord.DeleteMessagesParams{
		Messages: []disgord.Snowflake{disgord.ParseSnowflakeString(i.Args[0])},
	}

	// results are ordered newest to oldest, oldest being the 'from' id (arg[0])
	for i := len(results) - 1; i > 0; i-- {
		r := results[i]
		delParams.Messages = append(delParams.Messages, r.ID)
		if r.ID.String() == before {
			break
		}
	}

	if len(delParams.Messages) < 2 {
		return "Not enough messages to delete"
	}

	// add the command itself to the list of id's to delete
	delParams.Messages = append(delParams.Messages, sub.msg.ID)

	// perform the delete
	e = sub.sess.DeleteMessages(sub.ctx, sub.msg.ChannelID, delParams)
	if e != nil {
		return e.Error()
	}

	return ""
}

func pingBlock(s disgord.Session, m *disgord.MessageCreate) bool {
	mentions := m.Message.Mentions
	if len(mentions) == 0 {
		return false
	}

	block := false
	for _, user := range mentions {
		if index(admins, user.Username) != -1 {
			block = true
			break
		}
	}

	if !block {
		return false
	}

	e := s.DeleteMessage(m.Ctx, m.Message.ChannelID, m.Message.ID)
	if e != nil {
		log.Println(e)
		return true
	}

	message := m.Message.Content
	for _, user := range mentions {
		message = strings.Replace(message, user.Mention(), user.Username, -1)
	}

	_, e = s.SendMsg(m.Ctx, m.Message.ChannelID, "pls no ping")
	_, e = s.SendMsg(m.Ctx, m.Message.ChannelID, m.Message.Author.Username+": "+message)

	if e != nil {
		log.Println(e)
		return true
	}

	return true
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

func index(src []string, value string) int {
	for i, s := range src {
		if s == value {
			return i
		}
	}
	return -1
}

func name(array []*disgord.Role, value disgord.Snowflake) string {
	for _, v := range array {
		if v.ID == value {
			return v.Name
		}
	}
	return ""
}
