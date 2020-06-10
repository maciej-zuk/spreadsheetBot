package src

import (
	"fmt"
	"log"
)

// PerformAssign  -
func PerformAssign(ctx *RuntimeContext) error {
	hadErrors := false

	for _, cfg := range ctx.Configs {
		names, err := getCurrentAssignment(ctx, &cfg)
		if err != nil {
			log.Println("Error while loading assigned people from spreadsheet for group", cfg.GroupName, ":", err)
			hadErrors = true
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
		err = assignUsersToUserGroups(ctx, names, &cfg)
		if err != nil {
			log.Println("Error while assigning user group", cfg.GroupName, ":", err)
			hadErrors = true
			continue
		}
	}

	if hadErrors {
		return fmt.Errorf("Encountered errors during assignment")
	}
	return nil
}
