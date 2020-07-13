package discorddata

import (
	"sort"
	"strconv"
	"time"

	"emperror.dev/errors"
	"github.com/jonas747/discordgo"
	"github.com/jonas747/dutil"
	"github.com/jonas747/yagpdb/bot/botrest"
	"github.com/jonas747/yagpdb/common"
	"github.com/karlseguin/ccache"
	"golang.org/x/oauth2"
)

type Plugin struct{}

func (p *Plugin) PluginInfo() *common.PluginInfo {
	return &common.PluginInfo{
		Name:     "web_discorddata",
		SysName:  "web_discorddata",
		Category: common.PluginCategoryMisc,
	}
}

var logger = common.GetPluginLogger(&Plugin{})

func RegisterPlugin() {
	common.RegisterPlugin(&Plugin{})
}

var applicationCache = ccache.New(ccache.Configure().MaxSize(10000).ItemsToPrune(100))

func keySession(raw string) string {
	return "discord_session:" + raw
}

func GetSession(raw string, tokenDecoder func(string) (*oauth2.Token, error)) (*discordgo.Session, error) {
	result, err := applicationCache.Fetch(keySession(raw), time.Minute*10, func() (interface{}, error) {
		decoded, err := tokenDecoder(raw)
		if err != nil {
			return nil, errors.WithStackIf(err)
		}

		session, err := discordgo.New(decoded.Type() + " " + decoded.AccessToken)
		if err != nil {
			return nil, errors.WithStackIf(err)
		}

		return session, nil
	})
	if err != nil {
		return nil, err
	}

	return result.Value().(*discordgo.Session), nil
}

func keyUserInfo(token string) string {
	return "user_info_token:" + token
}

func GetUserInfo(token string, session *discordgo.Session) (*discordgo.User, error) {
	result, err := applicationCache.Fetch(keyUserInfo(token), time.Minute*10, func() (interface{}, error) {
		user, err := session.UserMe()
		if err != nil {
			return nil, errors.WithStackIf(err)
		}

		return user, nil
	})

	if err != nil {
		return nil, err
	}

	return result.Value().(*discordgo.User), nil
}

func keyFullGuild(guildID int64) string {
	return "full_guild:" + strconv.FormatInt(guildID, 10)
}

// GetFullGuild returns the guild from either:
// 1. Application cache
// 2. Botrest
// 3. Discord api
//
// It will will also make sure channels are included in the event we fall back to the discord API
func GetFullGuild(guildID int64) (*discordgo.Guild, error) {
	result, err := applicationCache.Fetch(keyFullGuild(guildID), time.Minute*10, func() (interface{}, error) {
		guild, err := botrest.GetGuild(guildID)
		if err != nil {
			// fall back to discord API
			guild, err = common.BotSession.Guild(guildID)
			if err != nil {
				return nil, err
			}

			// we also need to include channels as they're not included in the guild response
			channels, err := common.BotSession.GuildChannels(guildID)
			if err != nil {
				return nil, err
			}

			guild.Channels = channels
		}

		sort.Sort(dutil.Channels(guild.Channels))

		return guild, nil
	})

	if err != nil {
		return nil, err
	}

	return result.Value().(*discordgo.Guild), nil
}