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
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	pongo "github.com/flosch/pongo2"
	"github.com/gorilla/mux"

	"github.com/ContentMine/wikibase"
)

const WIKIBASE_URL = "http://sciencesource.wmflabs.org"
const SPARQL_QUERY_URL = "http://sciencesource-query.wmflabs.org/proxy/wdqs/bigdata/namespace/wdq/sparql"

const ARTICLE_LIST_QUERY_SPARQL = `
SELECT ?res ?page_ID ?article_text_title WHERE {
  ?res wdt:P3 wd:Q4.
  OPTIONAL { ?res wdt:P25 ?page_ID. }
  OPTIONAL { ?res wdt:P11 ?article_text_title. }
}
`

const ANNOTATION_LIST_QUERY_SPARQL = `
SELECT ?res ?ScienceSource_article_title ?term ?dictionary WHERE {
  ?res wdt:P12 wd:%s.
  ?annotation wdt:P19 ?res
  OPTIONAL { ?res wdt:P20 ?ScienceSource_article_title. }
  OPTIONAL { ?annotation wdt:P15 ?term. }
  OPTIONAL { ?annotation wdt:P16 ?dictionary. }
}
`

type ArticleInfo struct {
	Title  string
	PageID string
	ItemID string
}

type AnnotationInfo struct {
	Dictionary string
	Count      int
}

func (a ArticleInfo) RawItemID() string {
	return strings.TrimPrefix(a.ItemID, "http://sciencesource.wmflabs.org/entity/")
}

func getArticleList() ([]ArticleInfo, error) {

	resp, err := wikibase.MakeSPARQLQuery(SPARQL_QUERY_URL, ARTICLE_LIST_QUERY_SPARQL)
	if err != nil {
		return nil, err
	}

	data := make([]ArticleInfo, len(resp.Results.Bindings))
	i := 0
	for _, binding := range resp.Results.Bindings {

		a := ArticleInfo{
			Title:  binding["article_text_title"].Value,
			PageID: binding["page_ID"].Value,
			ItemID: binding["res"].Value,
		}
		data[i] = a
		i += 1
	}

	return data, nil
}

func homeHandler(w http.ResponseWriter, r *http.Request) {

	res, err := getArticleList()
	if err != nil {
		panic(err)
	}

	w.WriteHeader(http.StatusOK)

	t := pongo.Must(pongo.FromFile("templates/home.html"))
	err = t.ExecuteWriter(pongo.Context{"articles": res}, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getArticleAnnotationList(article_id string) (string, map[string]AnnotationInfo, error) {

	resp, err := wikibase.MakeSPARQLQuery(SPARQL_QUERY_URL, fmt.Sprintf(ANNOTATION_LIST_QUERY_SPARQL, article_id))
	if err != nil {
		return "", nil, err
	}

	title := ""
	annotations := make(map[string]AnnotationInfo, 0)

	for _, binding := range resp.Results.Bindings {
		if title == "" {
			title = binding["ScienceSource_article_title"].Value
		}
		term := binding["term"].Value
		if term == "" {
			continue
		}

		a, ok := annotations[term]
		if ok {
			a.Count += 1
		} else {
			a = AnnotationInfo{
				Dictionary: binding["dictionary"].Value,
				Count:      1,
			}
		}
		annotations[term] = a

	}

	return title, annotations, nil
}

func articleHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	title, annotations, err := getArticleAnnotationList(id)
	if err != nil {
		panic(err)
	}

	w.WriteHeader(http.StatusOK)

	t := pongo.Must(pongo.FromFile("templates/article.html"))
	err = t.ExecuteWriter(pongo.Context{"annotations": annotations, "title": title}, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func main() {

	r := mux.NewRouter()

	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/article/{id:Q[0-9]+}/", articleHandler)

	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:4242",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
