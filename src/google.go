package src

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-errors/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func getSheets(ctx *RuntimeContext) (*sheets.Service, error) {
	var srvc *sheets.Service = nil
	var err error = nil
	if ctx.GoogleAPIKey != "" {
		srvc, err = sheets.NewService(context.Background(), option.WithAPIKey(ctx.GoogleAPIKey))
	}
	if srvc == nil || err != nil {
		b, err := json.Marshal(ctx.GoogleCredentialsJSON)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets.readonly")
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		srvc, err = sheets.New(getClient(ctx, config))
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}
	return srvc, nil
}

func getClient(ctx *RuntimeContext, config *oauth2.Config) *http.Client {
	var token *oauth2.Token
	token, err := tokenFromFile(ctx)
	if err != nil {
		token = getTokenFromWeb(ctx, config)
		saveToken(ctx, token)
	}
	tokenSource := config.TokenSource(oauth2.NoContext, token)
	client := oauth2.NewClient(oauth2.NoContext, tokenSource)
	newToken, err := tokenSource.Token()
	if err == nil {
		saveToken(ctx, newToken)
	}
	return client
}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(ctx *RuntimeContext, config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	authCode, err := ctx.io.Prompt()
	if err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(ctx *RuntimeContext) (*oauth2.Token, error) {
	tok := oauth2.Token{}

	data, err := ctx.io.LoadBytes("token")
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &tok)
	return &tok, err
}

// Saves a token to a file path.
func saveToken(ctx *RuntimeContext, token *oauth2.Token) {
	b, err := json.Marshal(token)
	if err == nil {
		ctx.io.SaveBytes("token", b)
	}
}
