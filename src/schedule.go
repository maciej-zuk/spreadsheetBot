package src

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-errors/errors"
	"github.com/slack-go/slack"
)

const format = "Mon 2 Jan"

func getScheduleForDate(
	ctx *RuntimeContext,
	cfg *AssignmentsConfig,
	startDate time.Time,
	endDate time.Time,
	title string,
) (string, error) {
	var b strings.Builder
	schedule, err := getDailyAssignmentScheduleForDateRange(ctx, cfg, startDate, endDate)
	if err != nil {
		return "", errors.Wrap(err, 0)
	}
	fmt.Fprintln(&b, cfg.GroupName, title)
	for _, entry := range schedule {
		fmt.Fprintf(&b, "%s\t", entry.Date.Format(format))
		if len(entry.Names) > 0 {
			for n, name := range entry.Names {
				user := matchUserToName(ctx, name)
				if user != nil {
					entry.Names[n] = fmt.Sprintf("%s (%s @%s)", name, user.RealName, user.Name)
				} else {
					entry.Names[n] = fmt.Sprintf("%s (no Slack match!)", name)
				}
			}
			fmt.Fprint(&b, strings.Join(entry.Names, ", "))
		} else {
			if cfg.KeepWhenMissing {
				if entry.Date.Weekday() == time.Sunday || entry.Date.Weekday() == time.Saturday {
					continue
				}
				fmt.Fprint(&b, "*same as previous day*")
			} else {
				fmt.Fprint(&b, "*nobody is assigned*")
			}
		}
		fmt.Fprint(&b, "\n")
	}
	return b.String(), nil
}

func getScheduleForDateAsSlackBlocks(
	ctx *RuntimeContext,
	cfg *AssignmentsConfig,
	startDate time.Time,
	endDate time.Time,
	title string,
) ([]slack.Block, error) {
	blocks := make([]slack.Block, 0)
	schedule, err := getDailyAssignmentScheduleForDateRange(ctx, cfg, startDate, endDate)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	blocks = append(blocks, sectionBlockFor(fmt.Sprintln(cfg.GroupName, title)))
	for _, entry := range schedule {
		var assignmentsStr string
		if len(entry.Names) > 0 {
			for n, name := range entry.Names {
				user := matchUserToName(ctx, name)
				if user != nil {
					entry.Names[n] = fmt.Sprintf("<@%s>", user.ID)
				} else {
					entry.Names[n] = fmt.Sprintf("%s (no Slack match!)", name)
				}
			}
			assignmentsStr = strings.Join(entry.Names, ", ")
		} else {
			if cfg.KeepWhenMissing {
				if entry.Date.Weekday() == time.Sunday || entry.Date.Weekday() == time.Saturday {
					continue
				}
				assignmentsStr = "*same as previous day*"
			} else {
				assignmentsStr = "*nobody is assigned*"
			}
		}
		blocks = append(blocks, contextBlockFor(
			entry.Date.Format(format),
			assignmentsStr,
		))
	}
	return blocks, nil
}

// PrintScheduleForDateRange  -
func PrintScheduleForDateRange(ctx *RuntimeContext, startDate time.Time, endDate time.Time, title string) {
	for _, cfg := range ctx.Configs {
		s, err := getScheduleForDate(ctx, &cfg, startDate, endDate, title)
		if err == nil {
			fmt.Print(s)
		} else {
			fmt.Println("Unable to find assignments for", cfg.GroupName)
			fmt.Println(err)
		}
	}
}

// NotifySlackOfScheduleForDateRange  -
func NotifySlackOfScheduleForDateRange(ctx *RuntimeContext, startDate time.Time, endDate time.Time, title string) {
	for _, cfg := range ctx.Configs {
		var channelID string
		if channel := matchChannelToName(ctx, cfg.NotifyChannel); channel != nil {
			channelID = channel.ID
		} else {
			log.Println("Skipping missing or nonexisting channel for group", cfg.GroupName)
			continue
		}
		blocks, err := getScheduleForDateAsSlackBlocks(ctx, &cfg, startDate, endDate, title)
		if err != nil {
			log.Println("Unable to notify Slack about schedule for group", cfg.GroupName)
		}

		ctx.slack.JoinConversation(channelID)
		ctx.slack.SendMessage(
			channelID,
			slack.MsgOptionBlocks(blocks...),
		)
	}
}
