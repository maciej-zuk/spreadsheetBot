package main

import (
	"fmt"
	"log"
	"os"
	spbot "spbot/src"
	srclambda "spbot/srclambda"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

type spreadsheetBotEvent struct {
	Cmd          string `json:"command"`
	Ts           string `json:"timestamp"`
	Overlap      bool   `json:"overlap"`
	FilterGroups string `json:"filterGroups"`
}

func handleLambdaEvent(event spreadsheetBotEvent) error {
	io := srclambda.SSMIOStrategy{
		KeyPrefix: os.Getenv("SSM_KEY_PREFIX"),
	}
	ctx := spbot.CreateContext("config", &io)

	fmt.Println("Running command", event.Cmd, ", TS=", event.Ts, ", Overlap=", event.Overlap, ", FilterGroups=", event.FilterGroups)
	ts := time.Now()
	if event.Ts != "" {
		i, err := strconv.ParseInt(event.Ts, 10, 64)
		if err != nil {
			return err
		}
		ts = time.Unix(i, 0)
	}

	if event.Overlap {
		ctx.Overlap = true
	}

	if event.FilterGroups != "" {
		ctx.FilterGroups = event.FilterGroups
	}

	var (
		startDate time.Time
		endDate   time.Time
		title     string
	)

	switch event.Cmd {
	case "printSchedule", "notifySlack", "assignPagerDuty":
		startDate = ts.In(time.UTC).Truncate(7 * 24 * time.Hour)
		endDate = startDate.AddDate(0, 0, 5)
		title = "schedule for this week"
	case "printScheduleNextWeek", "notifySlackNextWeek", "assignPagerDutyNextWeek":
		startDate = ts.In(time.UTC).Truncate(7*24*time.Hour).AddDate(0, 0, 7)
		endDate = startDate.AddDate(0, 0, 5)
		title = "schedule for next week"
	case "printScheduleToday", "notifySlackToday":
		startDate = ts.In(time.UTC).Truncate(24 * time.Hour)
		endDate = startDate.AddDate(0, 0, 1)
		title = "schedule for today"
	}

	switch event.Cmd {
	case "printSchedule", "printScheduleNextWeek", "printScheduleToday":
		spbot.LoadSheets(ctx)
		spbot.LoadSlack(ctx)
		spbot.PrintScheduleForDateRange(ctx, startDate, endDate, title)
	case "notifySlack", "notifySlackNextWeek", "notifySlackToday":
		spbot.LoadSheets(ctx)
		spbot.LoadSlack(ctx)
		spbot.NotifySlackOfScheduleForDateRange(ctx, startDate, endDate, title)
	case "assignGroups":
		spbot.LoadSheets(ctx)
		spbot.LoadSlack(ctx)
		spbot.PerformAssign(ctx, time.Now())
	case "assignPagerDuty", "assignPagerDutyNextWeek":
		fmt.Println("Loading PD")
		spbot.LoadPagerduty(ctx)
		fmt.Println("Loading Sheets")
		spbot.LoadSheets(ctx)
		spbot.PagerDutyAssignTiers(ctx, startDate, endDate)
	case "verifySlackNames":
		spbot.LoadSheets(ctx)
		spbot.LoadSlack(ctx)
		spbot.VerifySlackNames(ctx)
	case "verifyPagerDutyNames":
		spbot.LoadSheets(ctx)
		spbot.LoadPagerduty(ctx)
		spbot.VerifyPagerDutyNames(ctx)
	default:
		log.Fatalln("No command specified")
	}

	return nil
}

func main() {
	lambda.Start(handleLambdaEvent)
}
