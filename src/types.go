package src

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

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

// AssignmentsConfig -
type AssignmentsConfig struct {
	SelectRange     string `json:"selectRange"`
	GroupName       string `json:"groupName"`
	NamesRow        int    `json:"namesRow"`
	SpreadsheetID   string `json:"spreadsheetID"`
	DatesCol        string `json:"datesCol"`
	KeepWhenMissing bool   `json:"keepWhenMissing"`
	NotifyUsers     bool   `json:"notifyUsers"`
	NotifyChannel   string `json:"notifyChannel"`
	AssignCharacter string `json:"assignCharacter"`
	namesRowNum     int
	datesColNum     int
	rowOffset       int
	colOffset       int
}

// RuntimeContext -
type RuntimeContext struct {
	Configs               []AssignmentsConfig `json:"configs"`
	GoogleAPIKey          string              `json:"googleAPIKey"`
	GoogleCredentialsJSON interface{}         `json:"googleCredentialsJson"`
	SlackBotAPIKey        string              `json:"slackBotAPIKey"`
	SlackAccessAPIKey     string              `json:"slackAccessAPIKey"`

	slack        *slack.Client
	slackP       *slack.Client
	sheets       *sheets.Service
	groups       UserGroupList
	users        UserList
	channels     ChannelList
	io           IOStrategy
	usersMatcher *closestmatch.ClosestMatch
}

// AssignmentsScheduleEntry  -
type AssignmentsScheduleEntry struct {
	Date  time.Time
	Names []string
}

// UserGroupList .
type UserGroupList []slack.UserGroup

// UserList .
type UserList []slack.User

// ChannelList .
type ChannelList []slack.Channel

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
func (a *CliIOStrategy) Save(name string, value string) error {
	return a.SaveBytes(name, []byte(value))
}

// Prompt -
func (a *CliIOStrategy) Prompt() (string, error) {
	var str string
	_, err := fmt.Scan(&str)
	return str, err
}
