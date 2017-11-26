package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
)

// Use https://github.com/shawnps/sessions.git@go17-context for go 1.7
// support until it is merged

var store *sessions.CookieStore

//go get -u github.com/jteeuwen/go-bindata/...
//go:generate go-bindata -pkg main -o bindata.go assets/... templates/...

func init() {
	gob.Register(Character{})
	gob.Register(&oauth2.Token{})
}

func main() {
	if err := configure(); err != nil {
		log.Fatalln("configure:", err)
	}

	store = sessions.NewCookieStore(sessionAuthKey)
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   0,
		HttpOnly: true,
	}

	router := mux.NewRouter()
	router.HandleFunc("/", Index).Methods("GET")
	router.PathPrefix("/assets/").HandlerFunc(StaticFiles).Methods("GET")
	router.HandleFunc("/authorize", Authorize).Methods("GET")
	router.HandleFunc("/logout", Logout).Methods("GET")

	var handler http.Handler = router
	handler = handlers.LoggingHandler(os.Stdout, handler)
	handler = ssoHandler(handler)
	log.Fatalln(http.ListenAndServe(httpAddr, handler))
}

func Authorize(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, sessionName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	code := r.FormValue("code")
	token, err := oauthConfig.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	session.Values["token"] = token

	client := oauthConfig.Client(r.Context(), token)
	resp, err := client.Get("https://login.eveonline.com/oauth/verify")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var obj Character
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	session.Values["character"] = obj
	if err := session.Save(r, w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func Index(w http.ResponseWriter, r *http.Request) {
	token, err := getToken(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	colonies, err := NewAPI(r.Context(), token).GetColonies(getCharacter(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderView(w, r, "index", nil, map[string]interface{}{
		"Colonies": colonies,
	})
}

func Logout(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, sessionName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	delete(session.Values, "character")
	if err := session.Save(r, w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func StaticFiles(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	info, err := AssetInfo(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	asset, _ := Asset(path)
	http.ServeContent(w, r, info.Name(), info.ModTime(), bytes.NewReader(asset))
}

func ssoHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/authorize" {
			h.ServeHTTP(w, r)
			return
		}

		session, err := store.Get(r, sessionName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		val := session.Values["character"]
		if character, ok := val.(Character); ok {
			h.ServeHTTP(w, setCharacter(r, character))
		} else {
			http.Redirect(w, r, oauthConfig.AuthCodeURL(""), http.StatusFound)
		}
	})
}

type _characterKey int

const characterKey _characterKey = 0

func setCharacter(r *http.Request, character Character) *http.Request {
	ctx := context.WithValue(r.Context(), characterKey, character)
	return r.WithContext(ctx)
}

func getCharacter(r *http.Request) Character {
	return r.Context().Value(characterKey).(Character)
}

func getToken(r *http.Request) (*oauth2.Token, error) {
	session, err := store.Get(r, sessionName)
	if err != nil {
		return nil, err
	}

	token, ok := session.Values["token"].(*oauth2.Token)
	if !ok {
		return nil, fmt.Errorf("token not a token")
	}

	return token, nil
}

func renderView(w http.ResponseWriter, r *http.Request, name string, helpers template.FuncMap, data interface{}) {
	funcs := template.FuncMap{
		"title": func(s string) string {
			return strings.Title(s)
		},
		"character": func() Character {
			return getCharacter(r)
		},
	}
	for k, v := range helpers {
		funcs[k] = v
	}
	t := template.New("template").Funcs(funcs)
	t = template.Must(t.Parse(loadTemplate("layout.html")))
	t = template.Must(t.Parse(loadTemplate(name + ".html")))
	if err := t.ExecuteTemplate(w, "base", data); err != nil {
		log.Println("error in renderView:", err)
	}
}

func loadTemplate(name string) string {
	bytes, err := Asset("templates/" + name)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}
