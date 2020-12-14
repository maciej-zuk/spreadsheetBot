package main

import (
	"flag"
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

	assignPagerDuty := flag.Bool("assignPagerDuty", false, "assign PagerDuty for this week")
	assignPagerDutyNextWeek := flag.Bool("assignPagerDutyNextWeek", false, "assign PagerDuty for next week")
	verifySlackNames := flag.Bool("verifySlackNames", false, "verify Slack <-> spreadsheet names")
	verifyPagerDutyNames := flag.Bool("verifyPagerDutyNames", false, "verify PagerDuty <-> spreadsheet names")

	verbose := flag.Bool("verbose", true, "increase output verbosity")

	flag.Parse()
	io := spbot.CliIOStrategy{}
	ctx := spbot.CreateContext(*configFile, &io)
	ctx.Verbose = *verbose

	var (
		startDate time.Time
		endDate   time.Time
		title     string
	)

	if *printSchedule || *notifySlack || *assignPagerDuty {
		startDate = time.Now().In(time.UTC).Truncate(7 * 24 * time.Hour)
		endDate = startDate.AddDate(0, 0, 5)
		title = "schedule for this week"
	}

	if *printScheduleNextWeek || *notifySlackNextWeek || *assignPagerDutyNextWeek {
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
		spbot.LoadSheets(ctx)
		spbot.LoadSlack(ctx)
		spbot.PrintScheduleForDateRange(ctx, startDate, endDate, title)
		return
	}

	if *notifySlack || *notifySlackNextWeek || *notifySlackToday {
		spbot.LoadSheets(ctx)
		spbot.LoadSlack(ctx)
		spbot.NotifySlackOfScheduleForDateRange(ctx, startDate, endDate, title)
		return
	}

	if *assignGroups {
		spbot.LoadSheets(ctx)
		spbot.LoadSlack(ctx)
		spbot.PerformAssign(ctx, time.Now())
		return
	}

	if *assignPagerDuty || *assignPagerDutyNextWeek {
		spbot.LoadPagerduty(ctx)
		spbot.LoadSheets(ctx)
		spbot.PagerDutyAssignTiers(ctx, startDate, endDate)
		return
	}

	if *verifySlackNames {
		spbot.LoadSheets(ctx)
		spbot.LoadSlack(ctx)
		spbot.VerifySlackNames(ctx)
		return
	}

	if *verifyPagerDutyNames {
		spbot.LoadSheets(ctx)
		spbot.LoadPagerduty(ctx)
		spbot.VerifyPagerDutyNames(ctx)
		return
	}

	flag.PrintDefaults()
}
