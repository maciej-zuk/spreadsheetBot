package src

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"math"
	"strings"
	"time"

	"github.com/sahilm/fuzzy"
	"github.com/slack-go/slack"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var snDateBase time.Time = time.Date(1899, time.December, 30, 0, 0, 0, 0, time.UTC)

func serialNumberToTime(sn float64) time.Time {
	days := math.Floor(sn)
	date := snDateBase.AddDate(0, 0, int(days))
	seconds := time.Duration(24 * 60 * 60 * (sn - days))
	return date.Add(seconds * time.Second)
}

func dateEqual(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func cleanUpName(name string) string {
	name = strings.ReplaceAll(name, "\n", " ")
	name = strings.ReplaceAll(name, "  ", " ")
	name = strings.TrimSpace(name)
	return name
}

func matchGroupToName(name string, groups []slack.UserGroup) *slack.UserGroup {
	possibleGroups := fuzzy.FindFrom(name, UserGroupList(groups))
	if len(possibleGroups) > 0 {
		return &groups[possibleGroups[0].Index]
	}
	return nil
}

func matchUserToName(name string, users []slack.User) *slack.User {
	possibleUsers := fuzzy.FindFrom(name, UserList(users))
	if len(possibleUsers) > 0 {
		return &users[possibleUsers[0].Index]
	}
	return nil
}

func matchChannelToName(name string, channels []slack.Channel) *slack.Channel {
	possibleChannels := fuzzy.FindFrom(name, ChannelList(channels))
	if len(possibleChannels) > 0 {
		return &channels[possibleChannels[0].Index]
	}
	return nil
}

func matchPrivChannelToName(name string, channels []slack.Group) *slack.Group {
	possibleChannels := fuzzy.FindFrom(name, PrivChannelList(channels))
	if len(possibleChannels) > 0 {
		return &channels[possibleChannels[0].Index]
	}
	return nil
}

// CreateContext -
func CreateContext(fileName string) (*RuntimeContext, error) {
	var runtimeContext RuntimeContext
	configFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(configFile, &runtimeContext)
	if err != nil {
		return nil, err
	}

	runtimeContext.sheets, err = sheets.NewService(context.Background(), option.WithAPIKey(runtimeContext.GoogleAPIKey))
	if err != nil {
		return nil, err
	}

	runtimeContext.slack = slack.New(runtimeContext.SlackAPIKey)

	runtimeContext.groups, err = runtimeContext.slack.GetUserGroups()
	if err != nil {
		return nil, err
	}

	runtimeContext.users, err = runtimeContext.slack.GetUsers()
	if err != nil {
		return nil, err
	}

	runtimeContext.channels, err = runtimeContext.slack.GetChannels(true)
	if err != nil {
		return nil, err
	}

	runtimeContext.privChannels, err = runtimeContext.slack.GetGroups(true)
	if err != nil {
		return nil, err
	}

	return &runtimeContext, nil
}
