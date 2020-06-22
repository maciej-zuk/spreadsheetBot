package src

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/go-errors/errors"
	"github.com/schollz/closestmatch"
	"github.com/slack-go/slack"
)

var snDateBase time.Time = time.Date(1899, time.December, 30, 0, 0, 0, 0, time.UTC)

// Stack -
func Stack(err error) string {
	if err == nil {
		return "(no error)"
	}
	stack, ok := err.(*errors.Error)
	if ok && stack != nil {
		return stack.ErrorStack()
	}
	return "(no stack)"
}

func colNoToName(n int) string {
	var b strings.Builder
	r := 0
	for n > 0 {
		n--
		n, r = n/26, n%26
		b.WriteByte(byte(r) + 'A')
	}
	return b.String()
}

func nameToColNo(name string) int {
	n := 0
	for _, c := range name {
		n = n*26 + 1 + int(c-'A')
	}
	return n
}

func nameToColRow(name string) (int, int) {
	col := 0
	row := 0
	offset := 0
	for i, c := range name {
		if c >= 'A' && c <= 'Z' {
			col = col*26 + 1 + int(c-'A')
		} else {
			offset = i
			break
		}
	}
	fmt.Sscanf(name[offset:], "%d", &row)
	return col, row
}

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

func matchGroupToName(ctx *RuntimeContext, name string) *slack.UserGroup {
	for _, group := range ctx.groups {
		if group.Handle == name {
			return &group
		}
	}
	return nil
}

func matchUserToName(ctx *RuntimeContext, name string) *slack.User {
	match := ctx.usersMatcher.Closest(name)
	for _, user := range ctx.users {
		if user.RealName == match {
			return &user
		}
	}
	return nil
}

func matchChannelToName(ctx *RuntimeContext, name string) *slack.Channel {
	for _, channel := range ctx.channels {
		if channel.Name == name {
			return &channel
		}
	}
	return nil
}

// VerifyNames -
func VerifyNames(ctx *RuntimeContext) {
	for _, cfg := range ctx.Configs {
		fmt.Println("Verifying names for", cfg.GroupName)
		names, err := getAllNames(ctx, &cfg)
		if err != nil {
			log.Println(Stack(err))
			log.Println("Unable to load names for", cfg.GroupName)
			continue
		}
		good := 0
		bad := 0
		for _, nameAndPos := range names {
			user := matchUserToName(ctx, nameAndPos.name)
			if user == nil {
				fmt.Printf("[%s:%d] %s -> unable to match!\n",
					colNoToName(nameAndPos.col),
					nameAndPos.row,
					nameAndPos.name,
				)
				bad++
			} else {
				fmt.Printf("[%s:%d] %s -> %s (@%s)\n",
					colNoToName(nameAndPos.col),
					nameAndPos.row,
					nameAndPos.name,
					user.RealName,
					user.Name,
				)
				good++
			}
		}
		fmt.Println(good, "matches,", bad, "missing")
	}
}

// CreateContext -
func CreateContext(fileName string, io IOStrategy) (*RuntimeContext, error) {
	var runtimeContext RuntimeContext
	runtimeContext.io = io
	configFile, err := io.LoadBytes(fileName)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	err = json.Unmarshal(configFile, &runtimeContext)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	for n, cfg := range runtimeContext.Configs {
		ranges := strings.Split(cfg.SelectRange, ":")
		startRangeCol, startRangeRow := nameToColRow(ranges[0])
		runtimeContext.Configs[n].namesRowNum = cfg.NamesRow - startRangeRow
		runtimeContext.Configs[n].datesColNum = nameToColNo(cfg.DatesCol) - startRangeCol
		runtimeContext.Configs[n].colOffset = startRangeCol
		runtimeContext.Configs[n].rowOffset = startRangeRow
	}

	runtimeContext.sheets, err = getSheets(&runtimeContext)
	if err != nil {
		return nil, err
	}

	runtimeContext.slack = slack.New(runtimeContext.SlackBotAPIKey, slack.OptionDebug(false))
	runtimeContext.slackP = slack.New(runtimeContext.SlackAccessAPIKey, slack.OptionDebug(false))

	runtimeContext.groups, err = runtimeContext.slack.GetUserGroups()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	runtimeContext.users, err = runtimeContext.slack.GetUsers()
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	runtimeContext.channels = make([]slack.Channel, 0)
	channelsCursor := ""
	for {
		channels, nextCursor, err := runtimeContext.slack.GetConversations(&slack.GetConversationsParameters{
			Cursor:          channelsCursor,
			ExcludeArchived: "true",
			Types:           []string{"public_channel", "private_channel"},
		})
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		runtimeContext.channels = append(runtimeContext.channels, channels...)
		if nextCursor != "" {
			channelsCursor = nextCursor
		} else {
			break
		}
	}

	userNames := make([]string, len(runtimeContext.users))
	for i, user := range runtimeContext.users {
		userNames[i] = user.RealName
	}
	runtimeContext.usersMatcher = closestmatch.New(userNames, []int{2, 3, 4, 5, 6})

	return &runtimeContext, nil
}
