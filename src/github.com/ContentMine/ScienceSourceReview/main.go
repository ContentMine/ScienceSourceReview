//   Copyright 2019 Content Mine Ltd
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package main

import (
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/mrjones/oauth"

	"github.com/ContentMine/wikibase"
)

var store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

type ServerConfig struct {
	Address              string                       `json:"address"`
	OAuthConsumer        wikibase.ConsumerInformation `json:"oauth"`
	WikibaseURL          string                       `json:"wikibase_url"`
	QueryServiceURL      string                       `json:"queryservice_url"`
	QueryServiceEmbedURL string                       `json:"queryservice_embed_url"`
}

type ServerContext struct {
	Configuration ServerConfig
	AccessToken   *oauth.AccessToken
	OAuthConsumer *oauth.Consumer
	CookieSession *sessions.Session
}

func init() {
	gob.Register(&oauth.RequestToken{})
	gob.Register(&oauth.AccessToken{})
}

func loadConfig(path string) (ServerConfig, error) {

	f, err := os.Open(path)
	if err != nil {
		return ServerConfig{}, err
	}

	var config ServerConfig
	err = json.NewDecoder(f).Decode(&config)
	return config, err
}

// Simple wrapper so we can provide server config to each call
type callWrapper struct {
	ServerConfig
	H func(*ServerContext, http.ResponseWriter, *http.Request)
}

func (cw callWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// We use the cookie session for storage as we're otherwise stateless, so may as well create
	// it here once rather than all over the code
	session, err := store.Get(r, "session-name")
	if err != nil {
		log.Printf("Error getting session: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// The OAuth consumer isn't thread safe, so we need to build one per request
	consumer := oauth.NewConsumer(
		cw.OAuthConsumer.Key,
		cw.OAuthConsumer.Secret,
		oauth.ServiceProvider{
			RequestTokenUrl:   fmt.Sprintf("%s/wiki/Special:OAuth/initiate", cw.WikibaseURL),
			AuthorizeTokenUrl: fmt.Sprintf("%s/wiki/Special:OAuth/authorize", cw.WikibaseURL),
			AccessTokenUrl:    fmt.Sprintf("%s/wiki/Special:OAuth/token", cw.WikibaseURL),
		})
	consumer.AdditionalAuthorizationUrlParams = map[string]string{
		"oauth_consumer_key": cw.OAuthConsumer.Key,
	}

	ctx := ServerContext{
		Configuration: cw.ServerConfig,
		OAuthConsumer: consumer,
		CookieSession: session,
	}

	v := session.Values["auth"]
	if t, ok := v.(*oauth.AccessToken); ok {
		ctx.AccessToken = t
	}

	cw.H(&ctx, w, r)
}

func main() {

	var config_path string
	flag.StringVar(&config_path, "config", "config.json", "configuration file, required")
	flag.Parse()

	config, err := loadConfig(config_path)
	if err != nil {
		panic(err)
	}
	log.Printf("config: %v", config)

	r := mux.NewRouter()

	r.Handle("/", callWrapper{config, homeHandler})
	r.Handle("/article/{id:Q[0-9]+}/", callWrapper{config, articleHandler})
	r.Handle("/article/{id:Q[0-9]+}/review/", callWrapper{config, reviewHandler})

	r.Handle("/auth/", callWrapper{config, authHandler})
	r.Handle("/token/", callWrapper{config, getTokenHandler})
	r.Handle("/deauth/", callWrapper{config, deauthHandler})

	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	srv := &http.Server{
		Handler:      r,
		Addr:         config.Address,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
