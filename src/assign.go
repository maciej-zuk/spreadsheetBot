package src

import (
	"log"
	"strings"
	"time"
)

// PerformAssign  -
func PerformAssign(ctx *RuntimeContext, date time.Time) {
	for _, cfg := range ctx.Configs {
		if len(ctx.FilterGroups) > 0 && !strings.Contains(ctx.FilterGroups, cfg.GroupName) {
			continue
		}

		names, err := getCurrentAssignment(ctx, cfg, date)
		if err != nil {
			log.Println("Error while loading assigned people from spreadsheet for group", cfg.GroupName, ":", err)
			log.Println(Stack(err))
			continue
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
			log.Println("Error while assigning user group", cfg.GroupName, ":", err)
			log.Println(Stack(err))
			continue
		}
	}
}
