### Example config:
```
{
  "configs": [
    {
      "selectRange": "A1:D11", // spreadsheet range taken into consideration, rows without dates in "datesCol" are ignored
      "groupName": "group", // name of the user group that will be assigned
      "notifyChannel": "channel", // channel for schedule notifications, no notification if omitted
      "namesRow": 1, // row with names list
      "datesCol": 1, // column with dates
      "spreadsheetID": "1VYs24HCPuWz4GVs1Q0rRyVDQI6QwURt8wPBEs9vY0io", // spreadsheet id 
      "keepWhenMissing": true, // if there is missing assignment for a day "true" will keep existing assignments rather than unassigning everyone
      "notifyUsers": true, // true - notify users in direct message about todays schedule (during "assignGroups" action)
      "assignCharacter": "o" // character that is expected to indicate actual assignment
    } ...
  ],
  "googleAPIKey": "XYZ",
  "slackAPIKey": "XYZ"
}
```

### namesRow/datesCol
Keep in mind that if `selectRange` starts from something other than `A1` you have to subtract some value from `namesRow`/`datesCol` eg. If you would normally have `datesCol=5` and `namesRow=3` but `selectRange` is updated to `C2` (3 cols, 2 row) you should correct `datesCol` to be 2 and `namesRow` to 2. Wrong setup will result in error message or simply no action.

### Spreadsheet ID
in this url: https://docs.google.com/spreadsheets/d/1VYs24HCPuWz4GVs1Q0rRyVDQI6QwURt8wPBEs9vY0io/ ID is `1VYs24HCPuWz4GVs1Q0rRyVDQI6QwURt8wPBEs9vY0io`.
This is also demo spreadsheet with expected format for example config.

### Required Slack app scopes:

Create Slack app here: https://api.slack.com/apps?new_app=1

```
channels:read
chat:write
groups:read
im:write
usergroups:read
usergroups:write
users:read
```

### Required Google API scopes:
Create Google project here https://console.developers.google.com/
API key should be enough to access read-only spreadsheets

```
../auth/spreadsheets.readonly
```

### Usage summary:
```
Usage of ./spbot:
  -assignGroups
    	assign Slack groups for schedule in spreadsheet
  -config string
    	config file (default "config.json")
  -notifySlack
    	notify Slack channels about schedule for this week
  -printSchedule
    	print textual schedule for this week
```
