package src

import (
	"fmt"
	"time"

	"google.golang.org/api/sheets/v4"
)

func getNamesForDate(cfg *AssignmentsConfig, results *sheets.ValueRange, date time.Time) ([]string, error) {
	selected := make([]string, 0)
	if cfg.NamesRow > len(results.Values) {
		return nil, fmt.Errorf("Names row not found within spreadsheet")
	}
	names := results.Values[cfg.NamesRow-1]
	for _, row := range results.Values {
		if cfg.DatesCol > len(row) {
			return nil, fmt.Errorf("Dates column not found within spreadsheet")
		}
		snString := row[cfg.DatesCol-1]
		if sn, ok := snString.(float64); ok {
			rowDate := serialNumberToTime(sn)
			if dateEqual(date, rowDate) {
				for colN, col := range row {
					if colString, ok := col.(string); ok && colString == cfg.AssignCharacter {
						name, ok := names[colN].(string)
						if ok {
							selected = append(selected, cleanUpName(name))
						}
					}
				}
			}
		}
	}
	return selected, nil
}

func getSpreadsheetData(ctx *RuntimeContext, cfg *AssignmentsConfig) (*sheets.ValueRange, error) {
	return ctx.sheets.
		Spreadsheets.
		Values.
		Get(cfg.SpreadsheetID, cfg.SelectRange).
		DateTimeRenderOption("SERIAL_NUMBER").
		ValueRenderOption("UNFORMATTED_VALUE").
		Do()
}

func getCurrentAssignment(ctx *RuntimeContext, cfg *AssignmentsConfig) ([]string, error) {
	result, err := getSpreadsheetData(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return getNamesForDate(cfg, result, time.Now())
}

func getAssignmentScheduleForWeek(ctx *RuntimeContext, cfg *AssignmentsConfig, date time.Time) ([]AssignmentsScheduleEntry, error) {
	schedule := make([]AssignmentsScheduleEntry, 7)
	result, err := getSpreadsheetData(ctx, cfg)
	if err != nil {
		return nil, err
	}
	weekStart := date.In(time.UTC).Truncate(7 * 24 * time.Hour)
	for d := 0; d < 7; d++ {
		dayDate := weekStart.AddDate(0, 0, d)
		names, err := getNamesForDate(cfg, result, dayDate)
		if err != nil {
			return nil, err
		}
		schedule[d] = AssignmentsScheduleEntry{
			Date:  dayDate,
			Names: names,
		}
	}
	return schedule, nil
}
