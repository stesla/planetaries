package main

import (
	"fmt"
	"github.com/gorilla/securecookie"
	"golang.org/x/oauth2"
	"os"
)

var (
	sessionAuthKey []byte
	sessionName    string
	oauthConfig    oauth2.Config
	httpAddr       string
)

func configure() error {
	if key, ok := os.LookupEnv("SESSION_AUTH_KEY"); ok {
		sessionAuthKey = []byte(key)
	} else {
		sessionAuthKey = securecookie.GenerateRandomKey(64)
	}
	sessionName = getEnvDefault("SESSION_NAME", "session")

	httpAddr = getEnvDefault("HTTP_ADDRESS", ":8080")

	oauthConfig = oauth2.Config{
		ClientID:     os.Getenv("OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("OAUTH_CLIENT_SECRET"),
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://login.eveonline.com/oauth/authorize",
			TokenURL: "https://login.eveonline.com/oauth/token",
		},
		RedirectURL: getEnvDefault("OAUTH_REDIRECT_URL", "http://localhost:8080/authorize"),
		Scopes: []string{
			"esi-planets.manage_planets.v1",
		},
	}
	if oauthConfig.ClientID == "" || oauthConfig.ClientSecret == "" {
		return fmt.Errorf("must provide both OAUTH_CLIENT_ID and OAUTH_CLIENT_SECRET")
	}

	return nil
}

func getEnvDefault(key string, defaultValue string) (result string) {
	var ok bool
	if result, ok = os.LookupEnv(key); !ok {
		result = defaultValue
	}
	return
}
