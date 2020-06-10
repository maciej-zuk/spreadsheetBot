package src

import (
	"time"

	"github.com/slack-go/slack"
	"google.golang.org/api/sheets/v4"
)

// AssignmentsConfig -

// AssignmentsConfig -
type AssignmentsConfig struct {
	SelectRange     string `json:"selectRange"`
	GroupName       string `json:"groupName"`
	NamesRow        int    `json:"namesRow"`
	SpreadsheetID   string `json:"spreadsheetID"`
	DatesCol        int    `json:"datesCol"`
	KeepWhenMissing bool   `json:"keepWhenMissing"`
	NotifyUsers     bool   `json:"notifyUsers"`
	NotifyChannel   string `json:"notifyChannel"`
	AssignCharacter string `json:"assignCharacter"`
}

// RuntimeContext -
type RuntimeContext struct {
	Configs      []AssignmentsConfig `json:"configs"`
	GoogleAPIKey string              `json:"googleAPIKey"`
	SlackAPIKey  string              `json:"slackAPIKey"`

	slack        *slack.Client
	sheets       *sheets.Service
	groups       UserGroupList
	users        UserList
	channels     ChannelList
	privChannels PrivChannelList
}

// AssignmentsScheduleEntry  -
type AssignmentsScheduleEntry struct {
	Date  time.Time
	Names []string
}

// UserGroupList .
type UserGroupList []slack.UserGroup

// String .
func (e UserGroupList) String(i int) string {
	return e[i].Name
}

// Len .
func (e UserGroupList) Len() int {
	return len(e)
}

// UserList .
type UserList []slack.User

// String .
func (e UserList) String(i int) string {
	return e[i].RealName
}

// Len .
func (e UserList) Len() int {
	return len(e)
}

// ChannelList .
type ChannelList []slack.Channel

// String .
func (e ChannelList) String(i int) string {
	return e[i].Name
}

// Len .
func (e ChannelList) Len() int {
	return len(e)
}

// PrivChannelList .
type PrivChannelList []slack.Group

// String .
func (e PrivChannelList) String(i int) string {
	return e[i].Name
}

// Len .
func (e PrivChannelList) Len() int {
	return len(e)
}
