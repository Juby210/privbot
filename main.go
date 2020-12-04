package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/diamondburned/arikawa/api"
	"github.com/diamondburned/arikawa/bot"
	"github.com/diamondburned/arikawa/bot/extras/middlewares"
	"github.com/diamondburned/arikawa/discord"
	"github.com/diamondburned/arikawa/gateway"
	"github.com/diamondburned/arikawa/utils/json/option"
)

type (
	cfg struct {
		Token, Prefix    string
		Selfroles        map[string]discord.RoleID
		Starboard        discord.ChannelID
		StarboardEnabled bool
	}

	hexColor string

	Bot struct {
		Ctx *bot.Context
	}
)

var config cfg

func (h *hexColor) Parse(arg string) error {
	if regexp.MustCompile(`^#?([a-fA-F0-9]{6})$`).MatchString(arg) {
		*h = hexColor(strings.ToLower(strings.TrimPrefix(arg, "#")))
	} else {
		return errors.New("The color must be in the hex format (#xxxxxx)")
	}
	return nil
}

func (h *hexColor) Usage() string {
	return "#xxxxxx"
}

func (bot *Bot) Setup(sub *bot.Subcommand) {
	sub.AddMiddleware("Clear", middlewares.GuildOnly(bot.Ctx))
	sub.AddMiddleware("Give", middlewares.GuildOnly(bot.Ctx))
	sub.AddMiddleware("Role", middlewares.GuildOnly(bot.Ctx))
}

func (bot *Bot) Help(*gateway.MessageCreateEvent) (string, error) {
	return bot.Ctx.Help(), nil
}

func (bot *Bot) Clear(m *gateway.MessageCreateEvent) (string, error) {
	for _, id := range m.Member.RoleIDs {
		r, _ := bot.Ctx.Role(m.GuildID, id)
		if strings.HasPrefix(r.Name, "color: ") {
			if roleMembersCount(bot.Ctx, m.GuildID, id) <= 1 {
				bot.Ctx.DeleteRole(m.GuildID, id)
			} else {
				bot.Ctx.RemoveRole(m.GuildID, m.Author.ID, id)
			}
		}
	}
	return "Cleared your color", nil
}

func (bot *Bot) Give(m *gateway.MessageCreateEvent, color hexColor) (string, error) {
	c := string(color)

	var cr *discord.Role
	for _, id := range m.Member.RoleIDs {
		r, _ := bot.Ctx.Role(m.GuildID, id)
		if strings.HasPrefix(r.Name, "color: #") {
			cr = r
		}
	}
	if cr != nil && cr.Name == "color: #"+c {
		return "Color added: `#" + c + "`", nil
	}

	roles, _ := bot.Ctx.Roles(m.GuildID)
	found := false
	for _, r := range roles {
		if r.Name == "color: #"+c {
			bot.Ctx.AddRole(m.GuildID, m.Author.ID, r.ID)
			found = true
		}
	}
	if !found {
		col, _ := strconv.ParseUint(c, 16, 32)
		r, _ := bot.Ctx.CreateRole(m.GuildID, api.CreateRoleData{Name: "color: #" + c, Color: discord.Color(col), Permissions: 0})

		var perm discord.Permissions = 0
		bot.Ctx.ModifyRole(m.GuildID, r.ID, api.ModifyRoleData{Permissions: &perm})
		mem, _ := bot.Ctx.Member(m.GuildID, bot.Ctx.Ready.User.ID)
		bot.Ctx.MoveRole(m.GuildID, []api.MoveRoleData{{ID: r.ID,
			Position: option.NewNullableInt(getHighestRolePos(bot.Ctx, m.GuildID, mem.RoleIDs))}})
		bot.Ctx.AddRole(m.GuildID, m.Author.ID, r.ID)
	}

	if cr != nil {
		if roleMembersCount(bot.Ctx, m.GuildID, cr.ID) <= 1 {
			bot.Ctx.DeleteRole(m.GuildID, cr.ID)
		} else {
			bot.Ctx.RemoveRole(m.GuildID, m.Author.ID, cr.ID)
		}
	}

	return "Color added: `#" + c + "`", nil
}

func (bot *Bot) Role(m *gateway.MessageCreateEvent, role string) (string, error) {
	id, ok := config.Selfroles[role]
	if !ok {
		roles := ""
		for r := range config.Selfroles {
			roles += "\n      " + r
		}
		return "__Selfroles__" + roles, nil
	}
	r, _ := bot.Ctx.Role(m.GuildID, id)
	for _, rID := range m.Member.RoleIDs {
		if rID == id {
			bot.Ctx.RemoveRole(m.GuildID, m.Author.ID, id)
			return "Removed role: `" + r.Name + "`", nil
		}
	}
	bot.Ctx.AddRole(m.GuildID, m.Author.ID, id)
	return "Added role: `" + r.Name + "`", nil
}

func roleMembersCount(ctx *bot.Context, gID discord.GuildID, rID discord.RoleID) (mr int) {
	m, _ := ctx.Members(gID)
	for _, mem := range m {
		for _, role := range mem.RoleIDs {
			if role == rID {
				mr++
			}
		}
	}
	return
}

func getHighestRolePos(ctx *bot.Context, gID discord.GuildID, roleIDs []discord.RoleID) (max int) {
	for _, r := range roleIDs {
		role, _ := ctx.Role(gID, r)
		if role.Position > max {
			max = role.Position
		}
	}
	return
}

func main() {
	var err error
	cfile := []byte(os.Getenv("CONFIG"))
	if string(cfile) == "" {
		cfile, err = ioutil.ReadFile("config.json")
		if err != nil {
			fmt.Println(err)
		}
	}

	err = json.Unmarshal(cfile, &config)
	if err != nil {
		log.Fatalln(err)
	}

	commands := &Bot{}

	wait, err := bot.Start(config.Token, commands, func(ctx *bot.Context) error {
		ctx.HasPrefix = bot.NewPrefix(config.Prefix)
		ctx.AddHandler(starboard(ctx))
		ctx.AddHandler(func(r *gateway.ReadyEvent) {
			fmt.Println("ready as " + r.User.Username)
			ctx.Gateway.UpdateStatus(gateway.UpdateStatusData{Game: &discord.Activity{Name: config.Prefix + "give (color)"}})
		})
		return nil
	})

	if err != nil {
		log.Fatalln(err)
	}

	if err := wait(); err != nil {
		log.Fatalln("Gateway fatal error:", err)
	}
}
