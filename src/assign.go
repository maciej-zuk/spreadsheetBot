package src

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// PerformAssign  -

// PerformAssign  -
func PerformAssign(ctx *RuntimeContext, date time.Time) {
	for _, cfg := range ctx.Configs {
		if len(ctx.FilterGroups) > 0 && !strings.Contains(ctx.FilterGroups, cfg.GroupName) {
			continue
		}

		names, err := getCurrentAssignment(ctx, cfg, date)
		if err != nil {
			ctx.io.Fatal(fmt.Sprintln("Error while loading assigned people from spreadsheet for group", cfg.GroupName, ":", err, Stack(err)))
		}
		if len(names) == 0 {
			// do not clear assignments on keepWhenMissing
			if cfg.KeepWhenMissing {
				continue
			} else {
				log.Println("Warn: No assignment for group", cfg.GroupName)
			}
		}
		err = assignUsersToUserGroups(ctx, names, cfg)
		if err != nil {
			ctx.io.Fatal(fmt.Sprintln("Error while assigning user group", cfg.GroupName, ":", err, Stack(err)))
		}
	}
}
