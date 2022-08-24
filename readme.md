### Example config:
```
{
  "configs": [
    {
      "selectRange": "A1:D11", // spreadsheet range taken into consideration, rows without dates in "datesCol" are ignored
      "groupName": "group", // name of the user group that will be assigned
      "notifyChannel": "channel", // channel for schedule notifications, no notification if omitted
      "namesRow": 1, // row with names list
      "datesCol": "A", // column with dates
      "spreadsheetID": "1VYs24HCPuWz4GVs1Q0rRyVDQI6QwURt8wPBEs9vY0io", // spreadsheet id 
      "keepWhenMissing": true, // if there is missing assignment for a day "true" will keep existing assignments rather than unassigning everyone
      "notifyUsers": true, // true - notify users in direct message about todays schedule (during "assignGroups" action)
      "assignCharacter": "o" // character that is expected to indicate actual assignment
    } ...
  ],
  "googleCredentials": {
    "client_id": "...",
    "project_id": "...",
    "client_secret": "..."
  }, // or "googleAPIKey"
  "slackAccessAPIKey": "xoxp-...",
  "slackBotAPIKey": "xoxb-..."
}
```

### Spreadsheet ID
in this url: https://docs.google.com/spreadsheets/d/1VYs24HCPuWz4GVs1Q0rRyVDQI6QwURt8wPBEs9vY0io/ ID is `1VYs24HCPuWz4GVs1Q0rRyVDQI6QwURt8wPBEs9vY0io`.
This is also demo spreadsheet with expected format for example config.

### Required Slack app scopes:

Create Slack app here: https://api.slack.com/apps?new_app=1
Due to Slack approach to user group management (ie. only owners/admins can modify it) it is required to have both user and bot scoped clients.

Bot scopes:
```
channels:read
channels:join
chat:write
groups:read
im:write
users:read
```
User scopes:
```
usergroups:read
usergroups:write
```
User scope should be created by workspace admin.

### Required Google API scopes:
Create Google project here https://console.developers.google.com/
API key (`googleAPIKey`) should be enough fo read-only access of globally accessible spreadsheets.

OAuth credentials has to be created for organization scoped spreadsheets.

Add Spreadsheets API and following OAuth scope:
```
../auth/spreadsheets.readonly
```

Add OAuth2 Web client ID with `http://localhost:9000/cb` added to `Authorised redirect URIs`.
Copy required fields to `googleCredentials`.

First run perform OAuth2 credentials exchange and create additional token file (CLI only).

### AWS Lambda:
It is possible to deploy app on AWS Lambda, just build `lambda.go` rather than `main.go`.
Lambda app will store sensitive data in SSM Parameter store, to use it create two params in SSM:

- `/bot_config_prefix/config` - with config json
- `/bot_config_prefix/token` - with token (authorize and generate using CLI)

then pass prefix (`/bot_config_prefix/` in this example) as `SSM_KEY_PREFIX` env variable to lambda function.

### Usage summary:
```
Usage of ./spbot:
  -assignGroups
      assign Slack groups for schedule in spreadsheet
  -config string
      config file (default "config.json")
  -notifySlack
      notify Slack channels about schedule for this week
  -notifySlackNextWeek
      notify Slack channels about schedule for next week
  -notifySlackToday
      notify Slack channels about schedule for today
  -printSchedule
      print textual schedule for this week
  -printScheduleNextWeek
      print textual schedule for next week
  -printScheduleToday
      print textual schedule for today
```
