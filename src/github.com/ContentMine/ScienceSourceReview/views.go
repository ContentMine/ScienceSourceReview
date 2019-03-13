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

	pongo "github.com/flosch/pongo2"
	"github.com/gorilla/mux"

	"github.com/ContentMine/wikibase"
)

/*const ARTICLE_LIST_QUERY_SPARQL = `
SELECT ?res ?page_ID ?article_text_title WHERE {
  ?res wdt:P3 wd:Q4.
  OPTIONAL { ?res wdt:P25 ?page_ID. }
  OPTIONAL { ?res wdt:P11 ?article_text_title. }
}
`*/
const ARTICLE_LIST_QUERY_SPARQL = `
SELECT ?res ?page_ID ?article_text_title WHERE {
  ?res wdt:P11 wd:Q2.
  OPTIONAL { ?res wdt:P12 ?page_ID. }
  OPTIONAL { ?res wdt:P4 ?article_text_title. }
}
`

/*const ANNOTATION_LIST_QUERY_SPARQL = `
SELECT ?res ?ScienceSource_article_title ?term ?dictionary WHERE {
  ?res wdt:P12 wd:%s.
  ?annotation wdt:P19 ?res
  OPTIONAL { ?res wdt:P20 ?ScienceSource_article_title. }
  OPTIONAL { ?annotation wdt:P15 ?term. }
  OPTIONAL { ?annotation wdt:P16 ?dictionary. }
}
`*/
const ANNOTATION_LIST_QUERY_SPARQL = `
SELECT ?res ?ScienceSource_article_title ?term ?dictionary WHERE {
  ?res wdt:P16 wd:%s.
  ?annotation wdt:P21 ?res
  OPTIONAL { ?res wdt:P2 ?ScienceSource_article_title. }
  OPTIONAL { ?annotation wdt:P18 ?term. }
  OPTIONAL { ?annotation wdt:P20 ?dictionary. }
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
	//return strings.TrimPrefix(a.ItemID, "http://sciencesource.wmflabs.org/entity/")
	return strings.TrimPrefix(a.ItemID, "http://wikibase.svc/entity/")
}

func getArticleList(queryservice_url string) ([]ArticleInfo, error) {

	resp, err := wikibase.MakeSPARQLQuery(queryservice_url, ARTICLE_LIST_QUERY_SPARQL)
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

func homeHandler(ctx *ServerContext, w http.ResponseWriter, r *http.Request) {

	res, err := getArticleList(ctx.Configuration.QueryServiceURL)
	if err != nil {
	    log.Printf("Error making query: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	t := pongo.Must(pongo.FromFile("templates/home.html"))
	err = t.ExecuteWriter(pongo.Context{"articles": res, "ctx": ctx}, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getArticleAnnotationList(queryservice_url string, article_id string) (string, map[string]AnnotationInfo, error) {

	resp, err := wikibase.MakeSPARQLQuery(queryservice_url, fmt.Sprintf(ANNOTATION_LIST_QUERY_SPARQL, article_id))
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

func articleHandler(ctx *ServerContext, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	title, annotations, err := getArticleAnnotationList(ctx.Configuration.QueryServiceURL, id)
	if err != nil {
	    log.Printf("Error making query: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

	t := pongo.Must(pongo.FromFile("templates/article.html"))
	err = t.ExecuteWriter(pongo.Context{"annotations": annotations, "title": title, "ctx": ctx}, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
