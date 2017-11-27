package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gregjones/httpcache"
	"golang.org/x/oauth2"
	"net/http"
)

var cache = httpcache.NewMemoryCacheTransport()

type API struct {
	ctx context.Context
	tok *oauth2.Token
}

func NewAPI(ctx context.Context, tok *oauth2.Token) *API {
	return &API{ctx, tok}
}

func (api *API) client() *http.Client {
	client := &http.Client{Transport: cache}
	ctx := context.WithValue(api.ctx, oauth2.HTTPClient, client)
	return oauthConfig.Client(ctx, api.tok)
}

func (api *API) Get(out interface{}, format string, parameters ...interface{}) (err error) {
	url := fmt.Sprintf(format, parameters...)
	resp, err := api.client().Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&out)
	return
}

type Character struct {
	ID   int    `json:"CharacterID"`
	Name string `json:"CharacterName"`
}

type Colony struct {
	SolarSystemId int    `json:"solar_system_id"`
	PlanetId      int    `json:"planet_id"`
	OwnerId       int    `json:"owner_id"`
	UpgradeLevel  int    `json:"upgrade_level"`
	NumberOfPins  int    `json:"num_pins"`
	LastUpdate    string `json:"last_update"`
	PlanetType    string `json:"planet_type"`

	api *API
}

func (api *API) GetColonies(char Character) (result []*Colony, err error) {
	err = api.Get(
		&result,
		"https://esi.tech.ccp.is/latest/characters/%v/planets/",
		char.ID)
	for i := range result {
		result[i].api = api
	}
	return
}

func (c Colony) Planet() (out Planet, err error) {
	err = c.api.Get(
		&out,
		"https://esi.tech.ccp.is/latest/universe/planets/%d/",
		c.PlanetId)
	return
}

type Planet struct {
	Id       int      `json:"planet_id"`
	Name     string   `json:"name"`
	TypeID   int      `json:"type_id"`
	Position Position `json:"position"`
	SystemID int      `json:"system_id"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}
