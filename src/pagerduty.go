package src

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/agnivade/levenshtein"
	"github.com/go-errors/errors"
)

var daysOfWeek []string = []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

// VerifyPagerDutyNames -
func VerifyPagerDutyNames(ctx *RuntimeContext) {
	for _, cfg := range ctx.Configs {
		if cfg.PagerDuty == nil {
			continue
		}

		fmt.Println("Verifying names for", cfg.GroupName)
		names, err := getAllNames(ctx, cfg)
		if err != nil {
			log.Println(Stack(err))
			log.Println("Unable to load names for", cfg.GroupName)
			continue
		}
		good := 0
		lq := 0
		bad := 0
		for _, nameAndPos := range names {
			user := matchPDUserToName(ctx, nameAndPos.name)
			if user == nil {
				fmt.Printf("[%s:%d] %s -> \033[0;31munable to match!\033[0m\n",
					colNoToName(nameAndPos.col),
					nameAndPos.row,
					nameAndPos.name,
				)
				bad++
			} else {
				dist := levenshtein.ComputeDistance(nameAndPos.name, user.Name)
				matchQuality := 100.0 - math.Min(100.0, math.Round(100.0*float64(dist)/float64(len(nameAndPos.name))))
				fmt.Printf("[%s:%d] %s -> %s (@%s), match: %.0f%%%s\n",
					colNoToName(nameAndPos.col),
					nameAndPos.row,
					nameAndPos.name,
					user.Name,
					user.APIObject.ID,
					matchQuality,
					(func() string {
						if matchQuality < 50 {
							lq++
							return " \033[0;31mwarning! low quality match\033[0m"
						}
						good++
						return ""
					})(),
				)
			}
		}
		fmt.Println(good, "matches,", lq, "low quality,", bad, "missing")
	}
}

// PagerDutyAssignTiers -
func PagerDutyAssignTiers(ctx *RuntimeContext, startDate, endDate time.Time) {
	err := pagerDutyAssignTiers(ctx, startDate, endDate)
	if err != nil {
		ctx.io.Fatal(Stack(err))
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func pagerDutyAssignTiers(ctx *RuntimeContext, startDate, endDate time.Time) error {
	now := time.Now()
	s1 := rand.NewSource(now.UnixNano())
	r1 := rand.New(s1)

	for _, cfg := range ctx.Configs {
		if cfg.PagerDuty == nil {
			continue
		}
		for _, pd := range cfg.PagerDuty {
			// load phase
			policyID := pd.PolicyID
			filterGroups := pd.Groups
			tierIDs := pd.TierIDs
			prefix := pd.Prefix

			fmt.Printf("Processing policy ID='%s' for group '%s'.\n", policyID, cfg.GroupName)

			// get schedule
			schedule, err := getDailyAssignmentScheduleForDateRange(ctx, cfg, startDate, endDate)
			if err != nil {
				return errors.Wrap(err, 0)
			}

			// init slots per group (for validation)
			slotsPerGroup := make(map[string]int)
			for _, group := range filterGroups {
				slotsPerGroup[group] = 0
			}

			// calculate slots per group and max slots per day
			maxPerDay := 0
			for _, entry := range schedule {
				currentMaxPerDay := 0
				for _, nameGroup := range entry.Names {
					slotsPerGroup[nameGroup.Group]++
					if filterGroups != nil && !contains(filterGroups, nameGroup.Group) {
						continue
					}
					currentMaxPerDay++
				}
				if currentMaxPerDay > maxPerDay {
					maxPerDay = currentMaxPerDay
				}
			}

			// verify max slots per day > tier capacity - fatal
			if len(tierIDs)*5 < maxPerDay {
				return errors.Errorf(
					"Schedule for policy id='%s' has up to %d people per day but it contains only %d tiers, %d tiers is required",
					policyID,
					maxPerDay,
					len(tierIDs),
					1+maxPerDay/5,
				)
			}

			// extend max slot so all tiers are covered
			if len(tierIDs) > maxPerDay {
				maxPerDay = len(tierIDs)
			}

			// create groups, copy named groups
			var groups []string = make([]string, 0, maxPerDay)
			for _, group := range filterGroups {
				if slotsPerGroup[group] > 0 {
					groups = append(groups, group)
				} else {
					fmt.Printf("Warning: empty group '%s', dropping.\n", group)
				}
			}

			// append backup groups
			i := len(groups)
			backupSlotsCount := 1
			for i < maxPerDay {
				groups = append(groups, fmt.Sprintf("Backup%d", backupSlotsCount))
				i++
				backupSlotsCount++
			}
			backupSlotsCount--

			// prepare assignment map
			assignments := make(map[string]*PagerDutyTierAssignment)
			for _, group := range groups {
				assignments[group] = &PagerDutyTierAssignment{
					Group:       group,
					Assignments: nil,
				}
			}

			// fill assignment map
			for _, entry := range schedule {
				dayOfWeek := uint(entry.Date.Weekday())
				r1.Shuffle(len(entry.Names), func(i, j int) { entry.Names[i], entry.Names[j] = entry.Names[j], entry.Names[i] })
				for _, nameGroup := range entry.Names {
					// skip users (filter by group)
					if filterGroups != nil && !contains(filterGroups, nameGroup.Group) {
						// fmt.Printf("Skipped '%s', group='%s'\n", nameGroup.Name, nameGroup.Group)
						continue
					}
					// match user to PD
					match := matchPDUserToName(ctx, nameGroup.Name)
					if match == nil {
						fmt.Printf("Unable to match user '%s' to PagerDuty user\n", nameGroup.Name)
						continue
					}

					location, err := time.LoadLocation(pd.Timezone)
					if err != nil {
						location = time.UTC
					}
					timeInLocal := time.Date(now.Year(), now.Month(), now.Day(), int(pd.Start), 0, 0, 0, location)

					// create assignment
					assignment := &PagerDutySlotAssignment{
						DayOfWeek: dayOfWeek,
						User:      fmt.Sprintf("%s|%s -> %s", match.APIObject.ID, nameGroup.Name, match.Name),
						StartUtc:  timeInLocal.In(time.UTC).Format("15:04:05"),
					}
					// try to assign to primary group
					moveToTier2 := false
					for _, assignment := range assignments[nameGroup.Group].Assignments {
						if assignment.DayOfWeek == dayOfWeek {
							moveToTier2 = true
							break
						}
					}
					// fallback to backup group
					if moveToTier2 {
						success := false
						for i := 0; i < backupSlotsCount; i++ {
							groupName := fmt.Sprintf("Backup%d", i+1)
							isFree := true
							for _, assignment := range assignments[groupName].Assignments {
								if assignment.DayOfWeek == dayOfWeek {
									isFree = false
									break
								}
							}
							if isFree {
								assignments[groupName].Assignments = append(
									assignments[groupName].Assignments,
									assignment,
								)
								success = true
								break
							}
						}
						if !success {
							fmt.Printf("Unable to assign user '%s' on %s - missing slots\n", nameGroup.Name, entry.Date.Format("Jan _2"))
						}
					} else {
						assignments[nameGroup.Group].Assignments = append(
							assignments[nameGroup.Group].Assignments,
							assignment,
						)
					}
				}
			}

			// distribute assignments across tiers
			tierAssignments := make([][]*PagerDutyTierAssignment, len(tierIDs))
			hadEmptyTier := false
			groupCnt := 0
			for n := range tierIDs {
				for i := 0; i < 5 && groupCnt < len(groups); i++ {
					tierAssignments[n] = append(tierAssignments[n], assignments[groups[groupCnt]])
					groupCnt++
				}
			}
			for i := 0; i < len(tierAssignments); i++ {
				hadEmptyTier = false
				for n, s := range tierAssignments {
					if len(s) != 0 {
						continue
					}
					hadEmptyTier = true
					maxLen := 0
					maxSliceIndex := 0
					for m, s2 := range tierAssignments {
						if len(s2) > maxLen {
							maxLen = len(s2)
							maxSliceIndex = m
						}
					}
					tierAssignments[n] = append(tierAssignments[n], tierAssignments[maxSliceIndex][maxLen-1])
					tierAssignments[maxSliceIndex] = tierAssignments[maxSliceIndex][:maxLen-1]
					break
				}

				if !hadEmptyTier {
					break
				}
			}

			if hadEmptyTier {
				return errors.Errorf("Unable to distribute assignments for policy id='%s'", policyID)
			}

			// get policy
			var opts pagerduty.GetEscalationPolicyOptions
			policy, err := ctx.pagerduty.GetEscalationPolicy(policyID, &opts)
			policy.Teams = nil

			if err != nil {
				return errors.Errorf("No policy id='%s'", policyID)
			}

			// clear old schedules from policy
			var oldSchedules []pagerduty.Schedule
			if !pd.Append {
				oldSchedules, err = clearAutoSchedules(ctx, policy, prefix)
				if err != nil {
					return err
				}
			}

			for _, group := range groups {
				assignments[group].SlotName = fmt.Sprintf("Slot_%s_%s%s_%06X", policy.Name, prefix, group, r1.Intn(1<<24))
			}

			fmt.Printf("Fetched policy '%s' and will perform following actions:\n", policy.Name)

			for n, tierID := range tierIDs {
				for m := range tierAssignments[n] {
					if pd.Append {
						fmt.Printf(
							"- append schedule for policy %s with %d layer(s) for group '%s' for rule ID='%s', with following user(s):\n",
							policy.Name,
							tierAssignments[n][m].SlotName,
							len(tierAssignments[n][m].Assignments),
							tierAssignments[n][m].Group,
							tierID,
						)
					} else {
						fmt.Printf(
							"- create schedule '%s' with %d layer(s) for group '%s' for rule ID='%s', with following user(s):\n",
							tierAssignments[n][m].SlotName,
							len(tierAssignments[n][m].Assignments),
							tierAssignments[n][m].Group,
							tierID,
						)
					}
					for l := range tierAssignments[n][m].Assignments {
						fmt.Printf(
							"\t- layer '%s'\tstarting at %s@UTC: \t%s\n",
							daysOfWeek[tierAssignments[n][m].Assignments[l].DayOfWeek],
							tierAssignments[n][m].Assignments[l].StartUtc,
							tierAssignments[n][m].Assignments[l].User,
						)
					}
				}
				fmt.Printf("- this will result in total %d schedule(s) per rule ID='%s'\n", len(tierAssignments[n]), tierID)
			}

			for n := range oldSchedules {
				fmt.Printf("- attempt to unassign and delete old schedule: %s (ID: '%s')\n", oldSchedules[n].Name, oldSchedules[n].ID)
			}

			if ctx.Verbose {
				fmt.Printf("Proceed? [y/N] ")

				reader := bufio.NewReader(os.Stdin)
				input, err := reader.ReadString('\n')
				if err != nil || len(input) < 1 || input[0] != 'y' {
					fmt.Println("Aborting.")
					continue
				}
			}

			fmt.Println("Proceeding")

			// execution phase

			if pd.Append {
				// append existing schedules
				for n, tierID := range tierIDs {
					err = appendTier(ctx, policy, tierID, tierAssignments[n], startDate, endDate, pd.Duration)
					if err != nil {
						return err
					}
				}

				fmt.Println("Policy updated - schedule appended")
			} else {
				// create new schedules
				for n, tierID := range tierIDs {
					err = fillTier(ctx, policy, tierID, tierAssignments[n], startDate, endDate, pd.Duration)
					if err != nil {
						return err
					}
				}

				// update policy
				_, err = ctx.pagerduty.UpdateEscalationPolicy(policyID, policy)
				if err != nil {
					return errors.Wrap(err, 0)
				}

				// remove old schedules
				for n := range oldSchedules {
					ctx.pagerduty.DeleteSchedule(oldSchedules[n].ID)
				}

				fmt.Println("Policy updated")
			}
		}
	}

	return nil
}

func clearAutoSchedules(ctx *RuntimeContext, policy *pagerduty.EscalationPolicy, prefix string) ([]pagerduty.Schedule, error) {
	var listOpts pagerduty.ListSchedulesOptions
	listOpts.Query = fmt.Sprintf("Slot_%s_%s", policy.Name, prefix)
	scheds, err := ctx.pagerduty.ListSchedules(listOpts)

	if err != nil {
		return nil, err
	}

	for n := range scheds.Schedules {
		deleteScheduleFromPolicy(policy, scheds.Schedules[n].ID)
	}

	return scheds.Schedules, nil
}

func deleteScheduleFromPolicy(policy *pagerduty.EscalationPolicy, scheduleID string) {
	for n, rule := range policy.EscalationRules {
		outTargets := make([]pagerduty.APIObject, 0, len(rule.Targets))
		for _, target := range rule.Targets {
			if target.Type == "schedule_reference" && scheduleID == target.ID {
				continue
			}
			outTargets = append(outTargets, target)
		}
		policy.EscalationRules[n].Targets = outTargets
	}
}

func fillTier(
	ctx *RuntimeContext,
	policy *pagerduty.EscalationPolicy,
	ruleID string,
	assignments []*PagerDutyTierAssignment,
	startDate time.Time,
	endDate time.Time,
	duration uint,
) error {
	var ruleNo = -1
	for n, rule := range policy.EscalationRules {
		if rule.ID == ruleID {
			ruleNo = n
		}
	}

	if ruleNo == -1 {
		return errors.Errorf("No rule id='%s'", ruleID)
	}

	policy.EscalationRules[ruleNo].Targets = make([]pagerduty.APIObject, len(assignments))
	for n, a := range assignments {
		schedule, err := createSlot(
			ctx,
			a.SlotName,
			a.Assignments,
			startDate,
			endDate,
			duration,
		)
		if err != nil {
			return err
		}
		policy.EscalationRules[ruleNo].Targets[n] = pagerduty.APIObject{
			ID:   schedule.ID,
			Type: "schedule_reference",
		}
	}
	return nil
}

func createSlot(
	ctx *RuntimeContext,
	slotName string,
	assignments []*PagerDutySlotAssignment,
	startDate time.Time,
	endDate time.Time,
	duration uint,
) (*pagerduty.Schedule, error) {
	var schedule pagerduty.Schedule
	schedule.Name = slotName
	schedule.TimeZone = "UTC"

	schedule.ScheduleLayers = make([]pagerduty.ScheduleLayer, len(assignments))

	for n, a := range assignments {
		schedule.ScheduleLayers[n].Name = daysOfWeek[a.DayOfWeek]
		schedule.ScheduleLayers[n].Start = startDate.Format(time.RFC3339)
		schedule.ScheduleLayers[n].End = endDate.Format(time.RFC3339)
		schedule.ScheduleLayers[n].RotationVirtualStart = startDate.Format(time.RFC3339)
		schedule.ScheduleLayers[n].RotationTurnLengthSeconds = 24 * 60 * 60
		schedule.ScheduleLayers[n].Users = make([]pagerduty.UserReference, 1)
		schedule.ScheduleLayers[n].Users[0].User = pagerduty.APIObject{
			ID:   strings.Split(a.User, "|")[0],
			Type: "user_reference",
		}
		schedule.ScheduleLayers[n].Restrictions = make([]pagerduty.Restriction, 1)
		schedule.ScheduleLayers[n].Restrictions[0].Type = "weekly_restriction"
		schedule.ScheduleLayers[n].Restrictions[0].StartTimeOfDay = a.StartUtc
		schedule.ScheduleLayers[n].Restrictions[0].DurationSeconds = duration * 60 * 60
		schedule.ScheduleLayers[n].Restrictions[0].StartDayOfWeek = a.DayOfWeek
	}

	if len(assignments) == 0 {
		schedule.Description = "Automatic schedule generator slot (placeholder only)"
		schedule.ScheduleLayers = make([]pagerduty.ScheduleLayer, 1)
		schedule.ScheduleLayers[0].Name = "Placeholder layer"
		schedule.ScheduleLayers[0].Start = startDate.Format(time.RFC3339)
		schedule.ScheduleLayers[0].End = startDate.Format(time.RFC3339)
		schedule.ScheduleLayers[0].RotationVirtualStart = startDate.Format(time.RFC3339)
		schedule.ScheduleLayers[0].RotationTurnLengthSeconds = 24 * 60 * 60
		schedule.ScheduleLayers[0].Users = make([]pagerduty.UserReference, 1)
		schedule.ScheduleLayers[0].Users[0].User = pagerduty.APIObject{
			ID:   ctx.pdUsers[0].APIObject.ID,
			Type: "user_reference",
		}
	} else {
		schedule.Description = "Automatic schedule generator slot (in use)"
	}

	savedSchedule, err := ctx.pagerduty.CreateSchedule(schedule)

	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return savedSchedule, err
}

func appendTier(
	ctx *RuntimeContext,
	policy *pagerduty.EscalationPolicy,
	ruleID string,
	assignments []*PagerDutyTierAssignment,
	startDate time.Time,
	endDate time.Time,
	duration uint,
) error {
	var ruleNo = -1
	for n, rule := range policy.EscalationRules {
		if rule.ID == ruleID {
			ruleNo = n
		}
	}

	if ruleNo == -1 {
		return errors.Errorf("No rule id='%s'", ruleID)
	}

	for n, a := range assignments {
		if policy.EscalationRules[ruleNo].Targets[n].Type != "schedule_reference" {
			return errors.Errorf("Policy ID='%s' at tier ID='%s', entry '%n' is not targetting a schedule", policy.ID, ruleID, n)
		}
		fmt.Println("Fetching schedule", policy.EscalationRules[ruleNo].Targets[n].ID)
		opts := pagerduty.GetScheduleOptions{}
		schedule, err := ctx.pagerduty.GetSchedule(policy.EscalationRules[ruleNo].Targets[n].ID, opts)
		if err != nil {
			return errors.Wrap(err, 0)
		}

		_, err = appendSlot(ctx, schedule, a.Assignments, startDate, endDate, duration)

		if err != nil {
			return err
		}

	}
	return nil
}

func appendSlot(
	ctx *RuntimeContext,
	schedule *pagerduty.Schedule,
	assignments []*PagerDutySlotAssignment,
	startDate time.Time,
	endDate time.Time,
	duration uint,
) (*pagerduty.Schedule, error) {
	for _, a := range assignments {
		layer := pagerduty.ScheduleLayer{
			Name:                      daysOfWeek[a.DayOfWeek],
			Start:                     startDate.Format(time.RFC3339),
			End:                       endDate.Format(time.RFC3339),
			RotationVirtualStart:      startDate.Format(time.RFC3339),
			RotationTurnLengthSeconds: 24 * 60 * 60,
			Users:                     make([]pagerduty.UserReference, 1),
			Restrictions:              make([]pagerduty.Restriction, 1),
		}
		layer.Users[0].User = pagerduty.APIObject{
			ID:   strings.Split(a.User, "|")[0],
			Type: "user_reference",
		}
		layer.Restrictions[0].Type = "weekly_restriction"
		layer.Restrictions[0].StartTimeOfDay = a.StartUtc
		layer.Restrictions[0].DurationSeconds = duration * 60 * 60
		layer.Restrictions[0].StartDayOfWeek = a.DayOfWeek
		schedule.ScheduleLayers = append(schedule.ScheduleLayers, layer)
	}

	savedSchedule, err := ctx.pagerduty.UpdateSchedule(schedule.ID, *schedule)

	if err != nil {
		return nil, errors.Wrap(err, 0)
	}

	return savedSchedule, err
}
