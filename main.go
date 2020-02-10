package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/andersfylling/disgord"
	"github.com/andersfylling/disgord/std"
)

type cfg struct {
	Token, Prefix string
	Selfroles     map[string]disgord.Snowflake
	Starboard     disgord.Snowflake
}

var (
	config cfg
	ctx    = context.Background()
)

func handler(s disgord.Session, data *disgord.MessageCreate) {
	msg := data.Message
	args := strings.Split(msg.Content, " ")
	g, _ := s.GetGuild(ctx, msg.GuildID)

	switch args[0] {
	case "give":
		if len(args) == 1 {
			msg.Reply(ctx, s, "Give role color hex (#xxxxxx)")
		} else {
			handleGive(msg, msg.Author.ID, args[1], s, g)
		}
	case "give2":
		p, _ := s.GetMemberPermissions(ctx, g.ID, msg.Author.ID)
		if p&disgord.PermissionAdministrator == 0 && p&disgord.PermissionManageRoles == 0 {
			msg.Reply(ctx, s, "You don't have permission to manage roles")
			return
		}

		if len(args) != 3 {
			msg.Reply(ctx, s, "Give role color hex (#xxxxxx) and mention user to give color")
		} else {
			if len(msg.Mentions) == 0 {
				msg.Reply(ctx, s, "Mention user to give color")
			} else {
				handleGive(msg, msg.Mentions[0].ID, args[1], s, g)
			}
		}
	case "clear":
		m, _ := g.Member(msg.Author.ID)

		for _, rID := range m.Roles {
			r, _ := g.Role(rID)
			if strings.HasPrefix(r.Name, "color: ") {
				if roleMembersCount(g, r.ID) <= 1 {
					s.DeleteGuildRole(ctx, g.ID, r.ID)
				} else {
					s.RemoveGuildMemberRole(ctx, g.ID, msg.Author.ID, r.ID)
				}
			}
		}
		msg.Reply(ctx, s, "Cleared your color")
	case "role":
		if len(args) > 1 {
			id, ok := config.Selfroles[args[1]]
			if ok {
				r, err := g.Role(id)
				if err != nil {
					msg.Reply(ctx, s, "Error: "+err.Error())
					return
				}

				m, _ := g.Member(msg.Author.ID)
				for _, rID := range m.Roles {
					if rID == id {
						s.RemoveGuildMemberRole(ctx, g.ID, msg.Author.ID, id)
						msg.Reply(ctx, s, "Removed role: `"+r.Name+"`")
						return
					}
				}

				s.AddGuildMemberRole(ctx, g.ID, msg.Author.ID, id)
				msg.Reply(ctx, s, "Added role: `"+r.Name+"`")
				return
			}
		}
		roles := ""
		for r := range config.Selfroles {
			if roles != "" {
				roles += ", "
			}
			roles += r
		}
		msg.Reply(ctx, s, "**Selfroles**: "+roles)
	}
}

func handleGive(msg *disgord.Message, uID disgord.Snowflake, carg string, s disgord.Session, g *disgord.Guild) {
	if regexp.MustCompile(`^#?([a-fA-F0-9]{6})$`).MatchString(carg) {
		color := strings.ToLower(strings.TrimPrefix(carg, "#"))

		m, _ := g.Member(uID)
		var cr *disgord.Role

		for _, rID := range m.Roles {
			r, _ := g.Role(rID)
			if strings.HasPrefix(r.Name, "color: #") {
				cr = r
			}
		}

		reply := func() {
			msg.Reply(ctx, s, "Color added: `#"+color+"`")
		}

		if cr != nil && cr.Name == "color: #"+color {
			reply()
			return
		}

		roles, _ := g.RoleByName("color: #" + color)
		if len(roles) >= 1 {
			s.AddGuildMemberRole(ctx, g.ID, uID, roles[0].ID)
		} else {
			col, _ := strconv.ParseUint(color, 16, 32)
			u, _ := s.GetCurrentUser(ctx)
			mem, _ := g.Member(u.ID)
			role, _ := s.CreateGuildRole(ctx, g.ID, &disgord.CreateGuildRoleParams{Name: "color: #" + color, Color: uint(col)})
			s.UpdateGuildRole(ctx, g.ID, role.ID).SetPermissions(0).Execute()
			s.UpdateGuildRolePositions(ctx, g.ID, []disgord.UpdateGuildRolePositionsParams{
				disgord.UpdateGuildRolePositionsParams{ID: role.ID, Position: getHighestRolePos(g, mem)}})
			s.AddGuildMemberRole(ctx, g.ID, uID, role.ID)
		}
		reply()

		if cr == nil {
			return
		}
		if roleMembersCount(g, cr.ID) <= 1 {
			s.DeleteGuildRole(ctx, g.ID, cr.ID)
		} else {
			s.RemoveGuildMemberRole(ctx, g.ID, uID, cr.ID)
		}
	} else {
		msg.Reply(ctx, s, "The color must be in the hex format")
	}
}

func roleMembersCount(g *disgord.Guild, rID disgord.Snowflake) int {
	mr := 0
	for _, mem := range g.Members {
		for _, role := range mem.Roles {
			if role == rID {
				mr++
			}
		}
	}
	return mr
}

func getHighestRolePos(g *disgord.Guild, mem *disgord.Member) int {
	max := -1
	for _, r := range mem.Roles {
		role, _ := g.Role(r)
		if role.Position > max {
			max = role.Position
		}
	}
	return max
}

func main() {
	cfile, err := ioutil.ReadFile("config.json")
	if err != nil {
		fmt.Println(err)
	}

	err = json.Unmarshal(cfile, &config)
	if err != nil {
		log.Fatalln(err)
	}

	client := disgord.New(disgord.Config{
		BotToken: config.Token,
		Logger:   disgord.DefaultLogger(false),
	})
	defer client.StayConnectedUntilInterrupted(ctx)

	filter, _ := std.NewMsgFilter(ctx, client)
	filter.SetPrefix(config.Prefix)

	client.On(disgord.EvtMessageCreate,
		filter.NotByBot,
		filter.HasPrefix,
		std.CopyMsgEvt,
		filter.StripPrefix,
		handler)

	client.On(disgord.EvtReady, func() {
		fmt.Println("ready")
		client.UpdateStatusString(config.Prefix + "give (color)")
	})

	client.On(disgord.EvtMessageReactionAdd, starboard)
}
