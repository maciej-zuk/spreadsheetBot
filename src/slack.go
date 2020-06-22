package src

import (
	"fmt"
	"log"
	"strings"

	"github.com/go-errors/errors"
	"github.com/slack-go/slack"
)

func assignUsersToUserGroups(ctx *RuntimeContext, names []string, cfg *AssignmentsConfig) error {
	userIds := make([]string, 0, len(names))

	fmt.Println("Assigning to group", cfg.GroupName)

	for _, name := range names {
		user := matchUserToName(ctx, name)
		if user == nil {
			fmt.Printf("Unable to match spreadsheet name '%s' to slack user\n", name)
			continue
		}
		fmt.Printf("%s -> %s (@%s)\n", name, user.RealName, user.Name)
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
