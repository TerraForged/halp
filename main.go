package main

import (
	"bytes"
	"context"
	"flag"
	"github.com/TerraForged/halp/cmd"
	"github.com/andersfylling/disgord"
	"log"
	"strings"
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

	log.Println("Loading command manager")
	commands := cmd.NewManager("commands.json")
	commands.Load()
	setup(commands)

	log.Println("Creating discord client")
	bot := disgord.New(disgord.Config{
		BotToken:           *token,
		Intents:            disgord.AllIntents(),
		ProjectName:        "halp",
		LoadMembersQuietly: true,
		RejectEvents: []string{
			// rarely used, and causes unnecessary spam
			disgord.EvtTypingStart,
			// these require special privilege
			// https://discord.com/developers/docs/topics/gateway#privileged-intents
			disgord.EvtPresenceUpdate,
			disgord.EvtGuildMemberAdd,
			disgord.EvtGuildMemberUpdate,
			disgord.EvtGuildMemberRemove,
		},
		Presence: &disgord.UpdateStatusPayload{
			Game: &disgord.Activity{
				Name: "!ping",
			},
		},
	})

	handle(bot, commands)

	log.Println("Connecting...")
	e := bot.Gateway().StayConnectedUntilInterrupted()

	if e != nil {
		panic(e)
	}

	log.Println("Shutting down")
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
	bot.Gateway().Ready(func(s disgord.Session, r *disgord.Ready) {
		log.Println("Setting status")
		e := s.UpdateStatusString("!list")
		if e != nil {
			log.Println(e)
		}
	})

	bot.Gateway().GuildCreate(func(s disgord.Session, g *disgord.GuildCreate) {
		log.Println("Joined guild:", g.Guild.Name)
	})

	bot.Gateway().MessageCreate(func(s disgord.Session, m *disgord.MessageCreate) {
		if m.Message.Author.Bot {
			return
		}

		if pingBlock(s, m.Message) {
			return
		}

		subject := &DiscordSubject{
			sess:  s,
			msg:   m.Message,
			ctx:   context.Background(),
			roles: nil,
			id:    m.Message.Author.ID,
			guild: m.Message.GuildID,
		}

		if success, message := commands.Process(subject, m.Message.Content); success && message != "" {
			_, e := s.SendMsg(m.Message.ChannelID, message)
			if e != nil {
				log.Println(e)
			}
		}
	})

	bot.Gateway().MessageUpdate(func(s disgord.Session, m *disgord.MessageUpdate) {
		if m.Message.Author.Bot {
			return
		}

		pingBlock(s, m.Message)
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
	results, e := sub.sess.Channel(sub.msg.ChannelID).GetMessages(params)
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
	e = sub.sess.Channel(sub.msg.ChannelID).DeleteMessages(delParams)
	if e != nil {
		return e.Error()
	}

	return ""
}

func pingBlock(s disgord.Session, m *disgord.Message) bool {
	mentions := m.Mentions
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

	e := s.Channel(m.ChannelID).Message(m.ID).Delete()
	if e != nil {
		log.Println(e)
		return true
	}

	header := "**From " + m.Author.Username + ":**"
	content := m.Content

	if m.MessageReference != nil {
		ref := m.MessageReference
		url := "https://discord.com/channels/" + ref.GuildID.String() + "/" + ref.ChannelID.String() + "/" + ref.MessageID.String()
		header = header + "\n_Replying to: <" + url + ">_"
	}

	for _, user := range mentions {
		content = strings.ReplaceAll(content, "<@"+user.ID.String()+">", user.Username)
		content = strings.ReplaceAll(content, "<@!"+user.ID.String()+">", user.Username)
	}

	_, e = s.SendMsg(m.ChannelID, "@ Mentions Removed!\n"+header+"\n"+content)
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

	m, e := s.sess.Guild(s.guild).Member(s.id).Get()
	if e != nil {
		log.Println(e)
		return nil
	}

	roles, e := s.sess.Guild(s.guild).GetRoles()
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
