package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/andersfylling/disgord"
	"github.com/andersfylling/disgord/std"
)

var (
	config map[string]interface{}
	ctx    = context.Background()
)

func handler(s disgord.Session, data *disgord.MessageCreate) {
	msg := data.Message
	args := strings.Split(msg.Content, " ")

	if args[0] == "give" {
		if len(args) == 1 {
			msg.Reply(ctx, s, "Give role color hex (#xxxxxx)")
		} else {
			handleGive(msg, msg.Author.ID, args[1], s)
		}
	} else if args[0] == "give2" {
		g, _ := s.GetGuild(ctx, msg.GuildID)
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
				handleGive(msg, msg.Mentions[0].ID, args[1], s)
			}
		}
	} else if args[0] == "clear" {
		g, _ := s.GetGuild(ctx, msg.GuildID)
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
	}
}

func handleGive(msg *disgord.Message, uID disgord.Snowflake, carg string, s disgord.Session) {
	validColor := regexp.MustCompile(`^#?([a-fA-F0-9]{6})$`)
	if validColor.MatchString(carg) {
		color := strings.ToLower(carg)
		if !strings.HasPrefix(color, "#") {
			color = "#" + color
		}

		g, _ := s.GetGuild(ctx, msg.GuildID)
		m, _ := g.Member(uID)
		var cr *disgord.Role

		for _, rID := range m.Roles {
			r, _ := g.Role(rID)
			if strings.HasPrefix(r.Name, "color: ") {
				cr = r
			}
		}
		if cr != nil && cr.Name == "color: "+color {
			s.AddGuildMemberRole(ctx, g.ID, uID, cr.ID)
			msg.Reply(ctx, s, "Color added "+color)
		} else {
			roles, _ := g.RoleByName("color: " + color)
			var role *disgord.Role
			if len(roles) >= 1 {
				role = roles[0]
			}
			msg.Reply(ctx, s, "Color added "+color)
			if role != nil {
				s.AddGuildMemberRole(ctx, g.ID, uID, role.ID)
			} else {
				color2 := strings.TrimPrefix(color, "#")
				col, _ := strconv.ParseUint(color2, 16, 32)
				u, _ := s.GetCurrentUser(ctx)
				mem, _ := g.Member(u.ID)
				role, _ = s.CreateGuildRole(ctx, g.ID, &disgord.CreateGuildRoleParams{Name: "color: " + color, Color: uint(col)})
				s.UpdateGuildRole(ctx, g.ID, role.ID).SetPermissions(0).Execute()
				s.UpdateGuildRolePositions(ctx, g.ID, []disgord.UpdateGuildRolePositionsParams{
					disgord.UpdateGuildRolePositionsParams{ID: role.ID, Position: getHighestRolePos(g, mem)}})
				s.AddGuildMemberRole(ctx, g.ID, uID, role.ID)
			}
			if cr == nil {
				return
			}
			if roleMembersCount(g, cr.ID) <= 1 {
				s.DeleteGuildRole(ctx, g.ID, cr.ID)
			} else {
				s.RemoveGuildMemberRole(ctx, g.ID, uID, cr.ID)
			}
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
	jsonFile, err := os.Open("config.json")
	if err != nil {
		fmt.Println(err)
	}
	defer jsonFile.Close()

	err = json.NewDecoder(jsonFile).Decode(&config)
	if err != nil {
		log.Fatalln(err)
	}

	client := disgord.New(disgord.Config{
		BotToken: config["token"].(string),
		Logger:   disgord.DefaultLogger(false),
	})
	defer client.StayConnectedUntilInterrupted(ctx)

	filter, _ := std.NewMsgFilter(ctx, client)
	filter.SetPrefix(config["prefix"].(string))

	client.On(disgord.EvtMessageCreate,
		filter.NotByBot,
		filter.HasPrefix,
		std.CopyMsgEvt,
		filter.StripPrefix,
		handler)

	client.On(disgord.EvtReady, func() {
		fmt.Println("ready")
		client.UpdateStatusString(config["prefix"].(string) + "give (color)")
	})
}
