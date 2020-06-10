package src

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

const format = "Mon 2 Jan"

func getWeekScheduleForDate(ctx *RuntimeContext, date time.Time) string {
	var b strings.Builder
	for _, cfg := range ctx.Configs {
		schedule, err := getAssignmentScheduleForWeek(ctx, &cfg, date)
		if err != nil {
			fmt.Fprintln(&b, "Unable to find assignments for", cfg.GroupName)
			continue
		}
		fmt.Fprintln(&b, "Assignment for group", cfg.GroupName)
		for _, entry := range schedule {
			fmt.Fprintf(&b, "%s\t", entry.Date.Format(format))
			if len(entry.Names) > 0 {
				fmt.Fprint(&b, strings.Join(entry.Names, ", "))
			} else {
				if cfg.KeepWhenMissing {
					fmt.Fprint(&b, "*same as previous day*")
				} else {
					fmt.Fprint(&b, "*nobody is assigned*")
				}
			}
			fmt.Fprint(&b, "\n")
		}
	}
	return b.String()
}

func getWeekScheduleForDateAsSlackBlocks(ctx *RuntimeContext, cfg *AssignmentsConfig, date time.Time) ([]slack.Block, error) {
	blocks := make([]slack.Block, 0, 8) // 7 days + header
	schedule, err := getAssignmentScheduleForWeek(ctx, cfg, date)
	if err != nil {
		return nil, err
	}
	blocks = append(blocks, sectionBlockFor(fmt.Sprintln("Assignment for group", cfg.GroupName)))
	for _, entry := range schedule {
		var assignmentsStr string
		if len(entry.Names) > 0 {
			for n, name := range entry.Names {
				user := matchUserToName(name, ctx.users)
				if user != nil {
					entry.Names[n] = fmt.Sprintf("<@%s>", user.ID)
				}
			}
			assignmentsStr = strings.Join(entry.Names, ", ")
		} else {
			if cfg.KeepWhenMissing {
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

// PrintWeekScheduleForDate  -
func PrintWeekScheduleForDate(ctx *RuntimeContext, date time.Time) {
	fmt.Print(getWeekScheduleForDate(ctx, date))
}

// NotifySlackOfWeekScheduleForDate  -
func NotifySlackOfWeekScheduleForDate(ctx *RuntimeContext, date time.Time) {
	for _, cfg := range ctx.Configs {
		var channelID string
		if channel := matchChannelToName(cfg.NotifyChannel, ctx.channels); channel != nil {
			channelID = channel.ID
		} else if channel := matchPrivChannelToName(cfg.NotifyChannel, ctx.privChannels); channel != nil {
			channelID = channel.ID
		} else {
			log.Println("Skipping missing or nonexisting channel for group", cfg.GroupName)
			continue
		}
		blocks, err := getWeekScheduleForDateAsSlackBlocks(ctx, &cfg, date)
		if err != nil {
			log.Println("Unable to notify Slack about schedule for group", cfg.GroupName)
		}

		ctx.slack.SendMessage(
			channelID,
			slack.MsgOptionBlocks(blocks...),
		)
	}
}
