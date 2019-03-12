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
    "encoding/json"
    "flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

type OauthToken struct {
    Key string `json:"key"`
    Secret string `json:"secret"`
}

type ServerConfig struct {
    Address string `json:"address"`
    OauthCredentials OauthToken `json:"oauth"`
    WikibaseURL string `json:"wikibase_url"`
    QueryServiceURL string `json:"queryservice_url"`
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
    *ServerConfig
    H func(*ServerConfig, http.ResponseWriter, *http.Request)
}

func (cw callWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    cw.H(cw.ServerConfig, w, r)
}

func main() {

	var config_path string
	flag.StringVar(&config_path, "config", "config.json", "configuration file, required")
	flag.Parse()

	config, err := loadConfig(config_path)
	if err != nil {
		panic(err)
	}

	r := mux.NewRouter()

	r.Handle("/", callWrapper{&config, homeHandler})
	r.Handle("/article/{id:Q[0-9]+}/", callWrapper{&config, articleHandler})

    r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	srv := &http.Server{
		Handler:      r,
		Addr:         config.Address,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
