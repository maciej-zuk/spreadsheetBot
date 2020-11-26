package src

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/schollz/closestmatch"
	"github.com/slack-go/slack"
	"google.golang.org/api/sheets/v4"
)

// IOStrategy -
type IOStrategy interface {
	Load(name string) (string, error)
	LoadBytes(name string) ([]byte, error)
	Save(name string, value string) error
	SaveBytes(name string, value []byte) error
	Prompt() (string, error)
}

// PagerDutySlotAssignment -
type PagerDutySlotAssignment struct {
	User      string
	StartUtc  string
	DayOfWeek uint
}

// PagerDutyTierAssignment -
type PagerDutyTierAssignment struct {
	Assignments []*PagerDutySlotAssignment
	Group       string
}

// NameGroup -
type NameGroup struct {
	Name  string
	Group string
}

// PagerDutyConfig -
type PagerDutyConfig struct {
	PolicyID string   `json:"policyID"`
	Groups   []string `json:"groups"`
	TierIDs  []string `json:"tierIDs"`
}

// AssignmentsConfig -
type AssignmentsConfig struct {
	PagerDuty       []*PagerDutyConfig `json:"pagerDuty"`
	SelectRange     string             `json:"selectRange"`
	GroupName       string             `json:"groupName"`
	SpreadsheetID   string             `json:"spreadsheetID"`
	DatesCol        string             `json:"datesCol"`
	NotifyChannel   string             `json:"notifyChannel"`
	AssignCharacter string             `json:"assignCharacter"`
	NamesRow        int                `json:"namesRow"`
	GroupsRow       int                `json:"groupsRow"`
	namesRowNum     int
	groupsRowNum    int
	datesColNum     int
	rowOffset       int
	colOffset       int
	KeepWhenMissing bool `json:"keepWhenMissing"`
	NotifyUsers     bool `json:"notifyUsers"`
}

// RuntimeContext -
type RuntimeContext struct {
	Configs               []*AssignmentsConfig `json:"configs"`
	GoogleCredentialsJSON interface{}          `json:"googleCredentialsJson"`
	GoogleAPIKey          string               `json:"googleAPIKey"`
	SlackBotAPIKey        string               `json:"slackBotAPIKey"`
	SlackAccessAPIKey     string               `json:"slackAccessAPIKey"`
	PagerDutyToken        string               `json:"pagerDutyToken"`

	slack          *slack.Client
	slackP         *slack.Client
	sheets         *sheets.Service
	groups         UserGroupList
	users          UserList
	pdUsers        PDUserList
	channels       ChannelList
	io             IOStrategy
	usersMatcher   *closestmatch.ClosestMatch
	pdUsersMatcher *closestmatch.ClosestMatch
	pagerduty      *pagerduty.Client
}

// AssignmentsScheduleEntry  -
type AssignmentsScheduleEntry struct {
	Date  time.Time
	Names []NameGroup
}

// UserGroupList .
type UserGroupList []slack.UserGroup

// UserList .
type UserList []slack.User

// ChannelList .
type ChannelList []slack.Channel

// PDUserList .
type PDUserList []pagerduty.User

// CliIOStrategy -
type CliIOStrategy struct{}

// LoadBytes -
func (a *CliIOStrategy) LoadBytes(name string) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}

// SaveBytes -
func (a *CliIOStrategy) SaveBytes(name string, value []byte) error {
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(value)
	return err
}

// Load -
func (a *CliIOStrategy) Load(name string) (string, error) {
	b, e := a.LoadBytes(name)
	return string(b), e
}

// Save -
func (a *CliIOStrategy) Save(name, value string) error {
	return a.SaveBytes(name, []byte(value))
}

// Prompt -
func (a *CliIOStrategy) Prompt() (string, error) {
	var str string
	_, err := fmt.Scan(&str)
	return str, err
}
