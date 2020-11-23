package src

import (
	"fmt"
	"log"
	"math"
	"math/rand"
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
		names, err := getAllNames(ctx, &cfg)
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
func PagerDutyAssignTiers(ctx *RuntimeContext, startDate time.Time, endDate time.Time) {
	err := pagerDutyAssignTiers(ctx, startDate, endDate)
	if err != nil {
		log.Fatalln(Stack(err))
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

func pagerDutyAssignTiers(ctx *RuntimeContext, startDate time.Time, endDate time.Time) error {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	for _, cfg := range ctx.Configs {
		if cfg.PagerDuty == nil {
			continue
		}
		for _, pd := range cfg.PagerDuty {
			policyID := pd.PolicyID
			filterGroups := pd.Groups
			tierIDs := pd.TierIDs

			// get schedule
			schedule, err := getDailyAssignmentScheduleForDateRange(ctx, &cfg, startDate, endDate)
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
				return errors.Errorf("Schedule for policy id='%s' has up to %d people per day but it contains only %d tiers, %d tiers is required", policyID, maxPerDay, len(tierIDs), 1+maxPerDay/5)
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
					// create assignment
					assignment := PagerDutySlotAssignment{
						DayOfWeek: dayOfWeek,
						User:      fmt.Sprintf("%s|%s -> %s", match.APIObject.ID, nameGroup.Name, match.Name),
						StartUtc:  "15:00:00",
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

			for _, group := range groups {
				fmt.Println(assignments[group])
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
					if len(s) == 0 {
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
				}

				if !hadEmptyTier {
					break
				}
			}

			if hadEmptyTier {
				return errors.Errorf("Unable to distribute assignments for policy id='%s'", policyID)
			}

			// fmt.Println(tierAssignments)
			// continue

			// get policy
			var opts pagerduty.GetEscalationPolicyOptions
			policy, err := ctx.pagerduty.GetEscalationPolicy(policyID, &opts)

			if err != nil {
				return errors.Errorf("No policy id='%s'", policyID)
			}

			// clear old schedules from policy
			oldSchedules, err := clearAutoSchedules(ctx, policy)

			if err != nil {
				return err
			}

			// create new schedules
			for n, tierID := range tierIDs {
				err = fillTier(ctx, policy, tierID, tierAssignments[n], startDate, endDate)
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
			for _, s := range oldSchedules {
				ctx.pagerduty.DeleteSchedule(s.ID)
			}
		}
	}

	return nil
}

func clearAutoSchedules(ctx *RuntimeContext, policy *pagerduty.EscalationPolicy) ([]pagerduty.Schedule, error) {
	var listOpts pagerduty.ListSchedulesOptions
	listOpts.Query = fmt.Sprintf("Slot_%s_", policy.Name)
	scheds, err := ctx.pagerduty.ListSchedules(listOpts)

	if err != nil {
		return nil, err
	}

	for _, s := range scheds.Schedules {
		deleteScheduleFromPolicy(ctx, policy, s.ID)
	}

	return scheds.Schedules, nil
}

func deleteScheduleFromPolicy(ctx *RuntimeContext, policy *pagerduty.EscalationPolicy, scheduleID string) {
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
) error {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

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
			fmt.Sprintf("%s_%s_%X", policy.Name, a.Group, r1.Intn(1<<16)),
			a.Assignments,
			startDate,
			endDate,
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
	group string,
	assignments []PagerDutySlotAssignment,
	startDate time.Time,
	endDate time.Time,
) (*pagerduty.Schedule, error) {
	var schedule pagerduty.Schedule
	schedule.Name = fmt.Sprintf("Slot_%s", group)
	schedule.TimeZone = "UTC"

	schedule.ScheduleLayers = make([]pagerduty.ScheduleLayer, len(assignments))

	for n, a := range assignments {
		schedule.ScheduleLayers[n].Name = daysOfWeek[a.DayOfWeek]
		schedule.ScheduleLayers[n].Start = startDate.Format(time.RFC3339)
		schedule.ScheduleLayers[n].End = endDate.Format(time.RFC3339)
		schedule.ScheduleLayers[n].RotationVirtualStart = startDate.Format(time.RFC3339)
		schedule.ScheduleLayers[n].RotationTurnLengthSeconds = 86400
		schedule.ScheduleLayers[n].Users = make([]pagerduty.UserReference, 1)
		schedule.ScheduleLayers[n].Users[0].User = pagerduty.APIObject{
			ID:   strings.Split(a.User, "|")[0],
			Type: "user_reference",
		}
		schedule.ScheduleLayers[n].Restrictions = make([]pagerduty.Restriction, 1)
		schedule.ScheduleLayers[n].Restrictions[0].Type = "weekly_restriction"
		schedule.ScheduleLayers[n].Restrictions[0].StartTimeOfDay = a.StartUtc
		schedule.ScheduleLayers[n].Restrictions[0].DurationSeconds = 28800
		schedule.ScheduleLayers[n].Restrictions[0].StartDayOfWeek = a.DayOfWeek
	}

	if len(assignments) == 0 {
		schedule.Description = "Automatic schedule generator slot (placeholder only)"
		schedule.ScheduleLayers = make([]pagerduty.ScheduleLayer, 1)
		schedule.ScheduleLayers[0].Name = "Placeholder layer"
		schedule.ScheduleLayers[0].Start = startDate.Format(time.RFC3339)
		schedule.ScheduleLayers[0].End = startDate.Format(time.RFC3339)
		schedule.ScheduleLayers[0].RotationVirtualStart = startDate.Format(time.RFC3339)
		schedule.ScheduleLayers[0].RotationTurnLengthSeconds = 86400
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
