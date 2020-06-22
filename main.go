package main

import (
	"flag"
	"log"
	spbot "spbot/src"
	"time"
)

func main() {
	configFile := flag.String("config", "config.json", "config file")

	assignGroups := flag.Bool("assignGroups", false, "assign Slack groups for schedule in spreadsheet")

	printSchedule := flag.Bool("printSchedule", false, "print textual schedule for this week")
	printScheduleToday := flag.Bool("printScheduleToday", false, "print textual schedule for today")
	printScheduleNextWeek := flag.Bool("printScheduleNextWeek", false, "print textual schedule for next week")

	notifySlack := flag.Bool("notifySlack", false, "notify Slack channels about schedule for this week")
	notifySlackToday := flag.Bool("notifySlackToday", false, "notify Slack channels about schedule for today")
	notifySlackNextWeek := flag.Bool("notifySlackNextWeek", false, "notify Slack channels about schedule for next week")

	flag.Parse()
	io := spbot.CliIOStrategy{}
	ctx, err := spbot.CreateContext(*configFile, &io)
	if err != nil {
		log.Fatalln(spbot.Stack(err))
	}

	var (
		startDate time.Time
		endDate   time.Time
		title     string
	)

	if *printSchedule || *notifySlack {
		startDate = time.Now().In(time.UTC).Truncate(7 * 24 * time.Hour)
		endDate = startDate.AddDate(0, 0, 5)
		title = "schedule for this week"
	}

	if *printScheduleNextWeek || *notifySlackNextWeek {
		startDate = time.Now().In(time.UTC).Truncate(7*24*time.Hour).AddDate(0, 0, 7)
		endDate = startDate.AddDate(0, 0, 5)
		title = "schedule for next week"
	}

	if *printScheduleToday || *notifySlackToday {
		startDate = time.Now().In(time.UTC).Truncate(24 * time.Hour)
		endDate = startDate.AddDate(0, 0, 1)
		title = "schedule for today"
	}

	if *printSchedule || *printScheduleNextWeek || *printScheduleToday {
		spbot.PrintScheduleForDateRange(ctx, startDate, endDate, title)
		return
	}

	if *notifySlack || *notifySlackNextWeek || *notifySlackToday {
		spbot.NotifySlackOfScheduleForDateRange(ctx, startDate, endDate, title)
		return
	}

	if *assignGroups {
		spbot.PerformAssign(ctx, time.Now())
		return
	}

	spbot.VerifyNames(ctx)

}
