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
	notifySlack := flag.Bool("notifySlack", false, "notify Slack channels about schedule for this week")

	flag.Parse()

	ctx, err := spbot.CreateContext(*configFile)
	if err != nil {
		log.Fatalln(err)
	}

	if *printSchedule {
		spbot.PrintWeekScheduleForDate(ctx, time.Now())

	}

	if *notifySlack {
		spbot.NotifySlackOfWeekScheduleForDate(ctx, time.Now())

	}

	if *assignGroups {
		err = spbot.PerformAssign(ctx)
		if err != nil {
			log.Fatalln(err)
		}
	}

}
