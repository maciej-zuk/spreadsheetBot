package src

import (
	"fmt"
	"strings"

	"github.com/slack-go/slack"
)

func assignUsersToUserGroups(ctx *RuntimeContext, names []string, cfg *AssignmentsConfig) error {
	targetGroup := matchGroupToName(cfg.GroupName, ctx.groups)
	if targetGroup == nil {
		return fmt.Errorf("Unable to match user group name '%s'", cfg.GroupName)
	}
	userIds := make([]string, 0, len(names))
	for _, name := range names {
		user := matchUserToName(name, ctx.users)
		if user == nil {
			fmt.Errorf("Unable to match spreadsheet name '%s' to slack user", name)
			continue
		}
		userIds = append(userIds, user.ID)
		notifyUserInGroup(ctx, user, cfg)
	}
	userIdsJoined := strings.Join(userIds, ",")
	_, err := ctx.slack.UpdateUserGroupMembers(targetGroup.ID, userIdsJoined)
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
