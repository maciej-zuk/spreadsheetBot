package src

import (
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/agnivade/levenshtein"
	"github.com/go-errors/errors"
	"github.com/slack-go/slack"
)

func assignUsersToUserGroups(ctx *RuntimeContext, names []NameGroup, cfg *AssignmentsConfig) error {
	userIds := make([]string, 0, len(names))

	fmt.Println("Assigning to group", cfg.GroupName)

	for _, name := range names {
		user := matchUserToName(ctx, name.Name)
		if user == nil {
			fmt.Printf("Unable to match spreadsheet name '%s' to slack user\n", name.Name)
			continue
		}
		fmt.Printf("%s -> %s (@%s)\n", name.Name, user.RealName, user.Name)
		userIds = append(userIds, user.ID)
		if cfg.NotifyUsers {
			notifyUserInGroup(ctx, user, cfg)
		}
	}

	targetGroup := matchGroupToName(ctx, cfg.GroupName)
	if targetGroup == nil {
		return errors.Errorf("Unable to match user group name '%s'", cfg.GroupName)
	}

	userIdsJoined := strings.Join(userIds, ",")
	_, err := ctx.slackP.UpdateUserGroupMembers(targetGroup.ID, userIdsJoined)
	return err
}

func notifyUserInGroup(ctx *RuntimeContext, user *slack.User, cfg *AssignmentsConfig) {
	channel, _, _, err := ctx.slack.OpenConversation(&slack.OpenConversationParameters{
		Users: []string{user.ID},
	})

	if err == nil {
		ctx.slack.SendMessage(
			channel.ID,
			slack.MsgOptionBlocks(sectionBlockFor(
				fmt.Sprintf("Hi there %s, a quick reminder for you: you have been assigned for *%s* group today!", user.RealName, cfg.GroupName),
			)),
		)
	} else {
		log.Println("Unable to notify", user.RealName)
	}
}

func sectionBlockFor(text string) slack.Block {
	return slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, text, false, false), nil, nil)
}

func contextBlockFor(text1, text2 string) slack.Block {
	return slack.NewContextBlock(
		"",
		slack.NewTextBlockObject(slack.MarkdownType, text1, false, false),
		slack.NewTextBlockObject(slack.MarkdownType, text2, false, false),
	)
}

// VerifySlackNames -
func VerifySlackNames(ctx *RuntimeContext) {
	for _, cfg := range ctx.Configs {
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
			user := matchUserToName(ctx, nameAndPos.name)
			if user == nil {
				fmt.Printf("[%s:%d] %s -> \033[0;31munable to match!\033[0m\n",
					colNoToName(nameAndPos.col),
					nameAndPos.row,
					nameAndPos.name,
				)
				bad++
			} else {
				dist := levenshtein.ComputeDistance(nameAndPos.name, user.RealName)
				matchQuality := 100.0 - math.Min(100.0, math.Round(100.0*float64(dist)/float64(len(nameAndPos.name))))
				fmt.Printf("[%s:%d] %s -> %s (@%s), match: %.0f%%%s\n",
					colNoToName(nameAndPos.col),
					nameAndPos.row,
					nameAndPos.name,
					user.RealName,
					user.Name,
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
