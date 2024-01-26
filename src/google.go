package src

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/go-errors/errors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type fullGoogleCredentials struct {
	ClientID                string   `json:"client_id"`
	ProjectID               string   `json:"project_id"`
	AuthUri                 string   `json:"auth_uri"`
	TokenUri                string   `json:"token_uri"`
	AuthProviderX509CertURL string   `json:"auth_provider_x509_cert_url"`
	ClientSecret            string   `json:"client_secret"`
	RedirectUris            []string `json:"redirect_uris"`
	Origins                 []string `json:"javascript_origins"`
}

func getSheets(ctx *RuntimeContext) (*sheets.Service, error) {
	var srvc *sheets.Service = nil
	var err error = nil
	if ctx.GoogleAPIKey != "" {
		if ctx.GoogleCredentials.ClientID != "" {
			fmt.Println("Warn: Both Google api key and Google credentials are present, Google api key takes precedence")
		}
		srvc, err = sheets.NewService(context.Background(), option.WithAPIKey(ctx.GoogleAPIKey))
	}
	if srvc == nil || err != nil {
		if ctx.GoogleCredentials.ClientID == "" || ctx.GoogleCredentials.ProjectID == "" || ctx.GoogleCredentials.ClientSecret == "" {
			return nil, errors.New("Both Google api key and Google credentials are missing ")
		}

		googleCredentialsJson := make(map[string]fullGoogleCredentials)
		googleCredentialsJson["installed"] = fullGoogleCredentials{
			ClientID:                ctx.GoogleCredentials.ClientID,
			ProjectID:               ctx.GoogleCredentials.ProjectID,
			ClientSecret:            ctx.GoogleCredentials.ClientSecret,
			AuthUri:                 "https://accounts.google.com/o/oauth2/auth",
			TokenUri:                "https://oauth2.googleapis.com/token",
			AuthProviderX509CertURL: "https://www.googleapis.com/oauth2/v1/certs",
			RedirectUris:            []string{"http://localhost:9000/cb"},
			Origins:                 []string{"http://localhost:9000"},
		}

		b, err := json.Marshal(googleCredentialsJson)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}

		config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets.readonly")
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		client, err := getClient(ctx, config)
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
		srvc, err = sheets.NewService(context.Background(), option.WithHTTPClient(client))
		if err != nil {
			return nil, errors.Wrap(err, 0)
		}
	}
	return srvc, nil
}

func getClient(ctx *RuntimeContext, config *oauth2.Config) (*http.Client, error) {
	var token *oauth2.Token
	token, err := tokenFromFile(ctx)
	if err != nil {
		token = getTokenFromWeb(ctx, config)
		err = saveToken(ctx, token)
		if err != nil {
			return nil, err
		}
	}
	tokenSource := config.TokenSource(context.Background(), token)
	client := oauth2.NewClient(context.Background(), tokenSource)
	newToken, err := tokenSource.Token()
	if err == nil {
		err = saveToken(ctx, newToken)
		if err != nil {
			return nil, err
		}
	}
	return client, err
}

var adhocServerClose chan bool
var adhocServerAuthCode string

func adhocAuthRedirect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	http.Redirect(w, r, ctx.Value("authURL").(string), 302)
}

func adhocAuthCb(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Done, you can close the window")
	adhocServerAuthCode = r.FormValue("code")
	adhocServerClose <- true
}

func openbrowser(ctx *RuntimeContext, url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		ctx.io.Fatal(fmt.Sprintf("%v", err))
	}

}

// Request a token from the web, then returns the retrieved token.
func getTokenFromWeb(ctx *RuntimeContext, config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	openbrowser(ctx, "http://localhost:9000/redirect")

	serverCtx := context.Background()
	adhocServerClose = make(chan bool)

	mux := http.NewServeMux()
	mux.HandleFunc("/redirect", adhocAuthRedirect)
	mux.HandleFunc("/cb", adhocAuthCb)
	adhocServer := &http.Server{
		Addr:    ":9000",
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			return context.WithValue(serverCtx, "authURL", authURL)
		},
	}

	go func() {
		adhocServer.ListenAndServe()
	}()

	<-adhocServerClose
	adhocServer.Shutdown(serverCtx)

	tok, err := config.Exchange(context.TODO(), adhocServerAuthCode)
	if err != nil {
		ctx.io.Fatal(fmt.Sprintf("Unable to retrieve token from web: %v", err))
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
func saveToken(ctx *RuntimeContext, token *oauth2.Token) error {
	b, err := json.Marshal(token)
	if err == nil {
		err = ctx.io.SaveBytes("token", b)
	}
	return err
}
