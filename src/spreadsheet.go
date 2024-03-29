package src

import (
	"time"

	"github.com/go-errors/errors"
	"google.golang.org/api/sheets/v4"
)

func getNamesForDate(cfg *AssignmentsConfig, results *sheets.ValueRange, date time.Time) ([]NameGroup, error) {
	selected := make([]NameGroup, 0)
	if cfg.namesRowNum < 0 || cfg.namesRowNum >= len(results.Values) {
		return nil, errors.Errorf("Names row not found within spreadsheet")
	}
	names := results.Values[cfg.namesRowNum]
	var groups []interface{}

	hasGroupRows := false
	if cfg.groupsRowNum >= 0 && cfg.groupsRowNum < len(results.Values) {
		hasGroupRows = true
		groups = results.Values[cfg.groupsRowNum]
	}

	for _, row := range results.Values {
		if cfg.datesColNum < 0 || cfg.datesColNum >= len(row) {
			return nil, errors.Errorf("Dates column not found within spreadsheet")
		}
		snString := row[cfg.datesColNum]
		if sn, ok := snString.(float64); ok {
			rowDate := serialNumberToTime(sn)
			if dateEqual(date, rowDate) {
				for colN, col := range row {
					if colString, ok := col.(string); ok && colString == cfg.AssignCharacter {
						name, ok := names[colN].(string)
						if ok {
							group := ""
							if hasGroupRows {
								group, _ = groups[colN].(string)
							}
							nameGroup := NameGroup{
								Name:  cleanUpName(name),
								Group: cleanUpName(group),
							}
							selected = append(selected, nameGroup)
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

func getCurrentAssignment(ctx *RuntimeContext, cfg *AssignmentsConfig, date time.Time) ([]NameGroup, error) {
	result, err := getSpreadsheetData(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	names, err := getNamesForDate(cfg, result, date)
	if err != nil {
		return nil, err
	}
	if ctx.Overlap {
		overlapDate := date.AddDate(0, 0, 1)
		if overlapDate.Weekday() == time.Saturday {
			overlapDate = overlapDate.AddDate(0, 0, 2)
		}
		overlapNames, err := getNamesForDate(cfg, result, overlapDate)
		if err != nil {
			return nil, err
		}
		uniqueMap := make(map[string]bool)
		for n := range names {
			uniqueMap[names[n].Name] = true
		}
		for n := range overlapNames {
			if _, ok := uniqueMap[overlapNames[n].Name]; !ok {
				names = append(names, overlapNames[n])
			}
		}
	}
	return names, err
}

func getDailyAssignmentScheduleForDateRange(
	ctx *RuntimeContext,
	cfg *AssignmentsConfig,
	startDate time.Time,
	endDate time.Time,
) ([]AssignmentsScheduleEntry, error) {
	schedule := make([]AssignmentsScheduleEntry, 0)
	result, err := getSpreadsheetData(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	for dayDate := startDate; dayDate.Before(endDate); dayDate = dayDate.AddDate(0, 0, 1) {
		names, err := getNamesForDate(cfg, result, dayDate)
		if err != nil {
			return nil, err
		}
		if ctx.Overlap {
			overlapDate := dayDate.AddDate(0, 0, 1)
			if overlapDate.Weekday() == time.Saturday {
				overlapDate = overlapDate.AddDate(0, 0, 2)
			}
			overlapNames, err := getNamesForDate(cfg, result, overlapDate)
			if err != nil {
				return nil, err
			}
			uniqueMap := make(map[string]bool)
			for n := range names {
				uniqueMap[names[n].Name] = true
			}
			for n := range overlapNames {
				if _, ok := uniqueMap[overlapNames[n].Name]; !ok {
					names = append(names, overlapNames[n])
				}
			}
		}
		schedule = append(schedule, AssignmentsScheduleEntry{
			Date:  dayDate,
			Names: names,
		})
	}
	return schedule, nil
}

type nameWithPos struct {
	name string
	col  int
	row  int
}

func getAllNames(ctx *RuntimeContext, cfg *AssignmentsConfig) ([]nameWithPos, error) {
	results, err := getSpreadsheetData(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err, 0)
	}
	if cfg.namesRowNum < 0 || cfg.namesRowNum >= len(results.Values) {
		return nil, errors.Errorf("Names row not found within spreadsheet")
	}
	names := results.Values[cfg.namesRowNum]
	cleanNames := make([]nameWithPos, 0, len(names))
	for i, name := range names {
		nameString, ok := name.(string)
		if ok {
			cleanName := cleanUpName(nameString)
			if i != cfg.datesColNum && cleanName != "" {
				cleanNames = append(cleanNames, nameWithPos{
					name: cleanName,
					col:  i + cfg.colOffset,
					row:  cfg.namesRowNum + cfg.rowOffset,
				})
			}
		}
	}

	return cleanNames, nil
}
