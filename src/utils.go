package src

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/agnivade/levenshtein"
	"github.com/go-errors/errors"
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

func nameToColRow(name string) (col, row int) {
	col = 0
	row = 0
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
	return
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
	for n := range ctx.groups {
		if ctx.groups[n].Handle == name {
			return &ctx.groups[n]
		}
	}
	return nil
}

func matchUserToName(ctx *RuntimeContext, name string) *slack.User {
	best := &ctx.users[0]
	bestDist := 9999999
	for n := range ctx.users {
		dist := levenshtein.ComputeDistance(ctx.users[n].RealName, name)
		if dist < bestDist {
			bestDist = dist
			best = &ctx.users[n]
		}
	}
	return best
}

func matchPDUserToName(ctx *RuntimeContext, name string) *pagerduty.User {
	best := &ctx.pdUsers[0]
	bestDist := 9999999
	for n := range ctx.pdUsers {
		dist := levenshtein.ComputeDistance(ctx.pdUsers[n].Name, name)
		if dist < bestDist {
			bestDist = dist
			best = &ctx.pdUsers[n]
		}
	}
	return best
}

func matchChannelToName(ctx *RuntimeContext, name string) *slack.Channel {
	channel, err := ctx.slack.GetConversationInfo(&slack.GetConversationInfoInput{
		ChannelID:         name,
		IncludeLocale:     false,
		IncludeNumMembers: false,
	})
	if err != nil {
		return nil
	}

	return channel
}

// CreateContext -
func CreateContext(fileName string, io IOStrategy) *RuntimeContext {
	var runtimeContext RuntimeContext
	runtimeContext.io = io
	runtimeContext.Verbose = false
	configFile, err := io.LoadBytes(fileName)
	if err != nil {
		log.Fatalln(Stack(errors.Wrap(err, 0)))
	}

	err = json.Unmarshal(configFile, &runtimeContext)
	if err != nil {
		log.Fatalln(Stack(errors.Wrap(err, 0)))
	}

	for n, cfg := range runtimeContext.Configs {
		var ranges []string
		if strings.Contains(cfg.SelectRange, "!") {
			actualRange := strings.Split(cfg.SelectRange, "!")[1]
			ranges = strings.Split(actualRange, ":")
		} else {
			ranges = strings.Split(cfg.SelectRange, ":")
		}
		startRangeCol, startRangeRow := nameToColRow(ranges[0])
		runtimeContext.Configs[n].namesRowNum = cfg.NamesRow - startRangeRow
		runtimeContext.Configs[n].groupsRowNum = cfg.GroupsRow - startRangeRow
		runtimeContext.Configs[n].datesColNum = nameToColNo(cfg.DatesCol) - startRangeCol
		runtimeContext.Configs[n].colOffset = startRangeCol
		runtimeContext.Configs[n].rowOffset = startRangeRow
	}
	return &runtimeContext
}

// LoadSheets -
func LoadSheets(ctx *RuntimeContext) {
	var err error
	ctx.sheets, err = getSheets(ctx)
	if err != nil {
		log.Fatalln(Stack(errors.Wrap(err, 0)))
	}
}

// LoadSlack -
func LoadSlack(ctx *RuntimeContext) {
	var err error
	ctx.slack = slack.New(ctx.SlackBotAPIKey, slack.OptionDebug(false))
	ctx.slackP = slack.New(ctx.SlackAccessAPIKey, slack.OptionDebug(false))

	ctx.groups, err = ctx.slack.GetUserGroups()
	if err != nil {
		log.Fatalln(Stack(errors.Wrap(err, 0)))
	}

	ctx.users, err = ctx.slack.GetUsers()
	if err != nil {
		log.Fatalln(Stack(errors.Wrap(err, 0)))
	}
}

// LoadPagerduty -
func LoadPagerduty(ctx *RuntimeContext) {
	ctx.pagerduty = pagerduty.NewClient(ctx.PagerDutyToken)

	var opts pagerduty.ListUsersOptions
	opts.Total = true
	opts.Limit = 1000
	users, err := ctx.pagerduty.ListUsers(opts)
	if err != nil {
		log.Fatalln(Stack(errors.Wrap(err, 0)))
	}
	ctx.pdUsers = users.Users
}
