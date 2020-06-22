package main

import (
	"fmt"
	"os"
	spbot "spbot/src"
	srclambda "spbot/srclambda"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
)

type spreadsheetBotEvent struct {
	Cmd string `json:"command"`
	Ts  string `json:"timestamp"`
}

func handleLambdaEvent(event spreadsheetBotEvent) error {
	io := srclambda.SSMIOStrategy{
		KeyPrefix: os.Getenv("SSM_KEY_PREFIX"),
	}
	ctx, err := spbot.CreateContext("config", &io)
	if err != nil {
		return err
	}

	fmt.Println("Running commnad", event.Cmd, ", TS=", event.Ts)
	ts := time.Now()
	if event.Ts != "" {
		i, err := strconv.ParseInt(event.Ts, 10, 64)
		if err != nil {
			return err
		}
		ts = time.Unix(i, 0)
	}

	var (
		startDate time.Time
		endDate   time.Time
		title     string
	)

	switch event.Cmd {
	case "printSchedule", "notifySlack":
		startDate = ts.In(time.UTC).Truncate(7 * 24 * time.Hour)
		endDate = startDate.AddDate(0, 0, 5)
		title = "schedule for this week"
	case "printScheduleNextWeek", "notifySlackNextWeek":
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
		spbot.PrintScheduleForDateRange(ctx, startDate, endDate, title)
	case "notifySlack", "notifySlackNextWeek", "notifySlackToday":
		spbot.NotifySlackOfScheduleForDateRange(ctx, startDate, endDate, title)
	case "assignGroups":
		spbot.PerformAssign(ctx, time.Now())
	default:
		spbot.VerifyNames(ctx)
	}

	return nil
}

func main() {
	lambda.Start(handleLambdaEvent)
}
