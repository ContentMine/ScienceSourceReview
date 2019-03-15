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
    "net/url"
	"strings"

	pongo "github.com/flosch/pongo2"
	"github.com/gorilla/mux"

	"github.com/ContentMine/wikibase"
)

/*const ARTICLE_LIST_QUERY_SPARQL = `
SELECT ?res ?page_ID ?article_text_title WHERE {
  ?res wdt:P3 wd:Q4.
  ?res wdt:P25 ?page_ID.
  ?res wdt:P11 ?article_text_title.
}
`*/
const ARTICLE_LIST_QUERY_SPARQL = `
SELECT ?res ?page_ID ?article_text_title WHERE {
  ?res wdt:P11 wd:Q2.
  ?res wdt:P12 ?page_ID.
  ?res wdt:P4 ?article_text_title.
}
`

/*const ANNOTATION_LIST_QUERY_SPARQL = `
SELECT ?res ?term ?dictionary ?Wikidata_item_code WHERE {
  ?res wdt:P12 wd:%s.
  ?annotation wdt:P19 ?res.
  ?annotation wdt:P15 ?term.
  ?annotation wdt:P16 ?dictionary.
  ?annotation wdt:P2 ?Wikidata_item_code.
}
`*/
const ANNOTATION_LIST_QUERY_SPARQL = `
SELECT ?res ?term ?dictionary ?Wikidata_item_code WHERE {
  ?res wdt:P16 wd:%s.
  ?annotation wdt:P21 ?res.
  ?annotation wdt:P18 ?term.
  ?annotation wdt:P20 ?dictionary.
  ?annotation wdt:P3 ?Wikidata_item_code.
}
`
/*const GRAPH_SPARQL = `
#defaultView:Dimensions
SELECT  ?drugLabel ?charnumber2 ?charnumber1 ?diseaseLabel
WHERE {
         ?anchor1 wdt:P12 wd:%s;
                  wdt:P10 ?charnumber1.
         ?anchor2 wdt:P12 wd:%s;
                  wdt:P10 ?charnumber2.
         ?term1 wdt:P19 ?anchor1.
         ?term2 wdt:P19 ?anchor2.
         ?term1 wdt:P15 ?disease.
         ?term2 wdt:P15 ?drug.
         ?term1 wdt:P16 "%s".
         ?term2 wdt:P16 "%s".
         FILTER (xsd:integer(?charnumber2) > xsd:integer(?charnumber1))
         FILTER (xsd:integer(?charnumber2) - xsd:integer(?charnumber1) < 200)

  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}
`*/
const GRAPH_SPARQL = `#defaultView:Dimensions
SELECT  ?drugLabel ?charnumber2 ?charnumber1 ?diseaseLabel
WHERE {
         ?anchor1 wdt:P16 wd:%s;
                  wdt:P7 ?charnumber1.
         ?anchor2 wdt:P16 wd:%s;
                  wdt:P7 ?charnumber2.
         ?term1 wdt:P21 ?anchor1.
         ?term2 wdt:P21 ?anchor2.
         ?term1 wdt:P18 ?disease.
         ?term2 wdt:P18 ?drug.
         ?term1 wdt:P20 "%s".
         ?term2 wdt:P20 "%s".
         FILTER (xsd:integer(?charnumber2) > xsd:integer(?charnumber1))
         FILTER (xsd:integer(?charnumber2) - xsd:integer(?charnumber1) < 200)

  SERVICE wikibase:label { bd:serviceParam wikibase:language "en". }
}`

const GET_ITEM_PROPERTIES_SPARQL = `
SELECT ?propUrl ?propLabel ?valUrl
WHERE
{
	hint:Query hint:optimizer 'None' .
	{	BIND(wd:%s AS ?valUrl) .
		BIND("N/A" AS ?propUrl ) .
		BIND("identity"@en AS ?propLabel ) .
	}
	UNION
	{	wd:%s ?propUrl ?valUrl .
		?property ?ref ?propUrl .
		?property rdf:type wikibase:Property .
		?property rdfs:label ?propLabel
	}
}
ORDER BY ?propUrl ?valUrl
`

// const TITLE_PROPERTY = "http://wikibase.svc/prop/direct/P11"
// const PAGEID_PROPERTY = "http://wikibase.svc/prop/direct/P25"
// const WIKIDATAID_PROPERTY = "http://wikibase.svc/prop/direct/P2"
const TITLE_PROPERTY = "http://wikibase.svc/prop/direct/P4"
const PAGEID_PROPERTY = "http://wikibase.svc/prop/direct/P12"
const WIKIDATAID_PROPERTY = "http://wikibase.svc/prop/direct/P3"

// const CLAIM_PROPERTY_ID = "P26"
const CLAIM_PROPERTY_ID = "P22"


type ArticleInfo struct {
	Title  string
	PageID string
	ItemID string
}

type AnnotationInfo struct {
    WikidataID string
	Dictionary string
	Count      int
}

type TermInfo struct {
    Label string
    WikidataID string
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

func getArticleProperties(queryservice_url string, article_id string) (map[string]string, error) {

	resp, err := wikibase.MakeSPARQLQuery(queryservice_url, fmt.Sprintf(GET_ITEM_PROPERTIES_SPARQL, article_id, article_id))
	if err != nil {
		return nil, err
	}

    res := make(map[string]string, len(resp.Results.Bindings))
    for _, binding := range resp.Results.Bindings {
        propUrl := binding["propUrl"].Value
        value := binding["valUrl"].Value
        if propUrl != "" && value != "" {
            res[propUrl] = value
        }
    }

    return res, nil
}

func getArticleAnnotationList(queryservice_url string, article_id string) (map[string]AnnotationInfo, error) {

	resp, err := wikibase.MakeSPARQLQuery(queryservice_url, fmt.Sprintf(ANNOTATION_LIST_QUERY_SPARQL, article_id))
	if err != nil {
		return nil, err
	}

	annotations := make(map[string]AnnotationInfo, 0)

	for _, binding := range resp.Results.Bindings {

		term := binding["term"].Value
		if term == "" {
			continue
		}

		a, ok := annotations[term]
		if ok {
			a.Count += 1
		} else {
			a = AnnotationInfo{
			    WikidataID: binding["Wikidata_item_code"].Value,
				Dictionary: binding["dictionary"].Value,
				Count:      1,
			}
		}
		annotations[term] = a

	}

	return annotations, nil
}

func articleHandler(ctx *ServerContext, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	properties, err := getArticleProperties(ctx.Configuration.QueryServiceURL, id)
	if err != nil {
	    log.Printf("Error making property query: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	title := properties[TITLE_PROPERTY]
	page_id := properties[PAGEID_PROPERTY]
	wikidata_id := properties[WIKIDATAID_PROPERTY]

	article_page_url := fmt.Sprintf("%s/?curid=%s", ctx.Configuration.WikibaseURL, page_id)
	scisource_page_url := fmt.Sprintf("%s/wiki/item:%s", ctx.Configuration.WikibaseURL, id)
	wikidata_page_url := fmt.Sprintf("https://wikidata.org/wiki/item:%s", wikidata_id)

	annotations, err := getArticleAnnotationList(ctx.Configuration.QueryServiceURL, id)
	if err != nil {
	    log.Printf("Error making annotation query: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

    // Can we extract the dictionary names?
    disease_dictionary := ""
    drug_dictionary := ""
    graph_sparql := ""

    // This is a bit of poor guesswork - in future the data model should support his better
    drugs := make([]TermInfo, 0)
    diseases := make([]TermInfo, 0)
    for key, value := range annotations {
        dict := value.Dictionary
        if strings.Contains(dict, "drug") {
            drug_dictionary = dict
            drugs = append(drugs, TermInfo{Label: key, WikidataID: value.WikidataID})
        } else {
            disease_dictionary = dict
            diseases = append(diseases, TermInfo{Label: key, WikidataID: value.WikidataID})
        }
    }

    if (disease_dictionary != "") && (drug_dictionary != "") {
    	encoded := url.PathEscape(fmt.Sprintf(GRAPH_SPARQL, id, id, disease_dictionary, drug_dictionary))
    	encoded = strings.ReplaceAll(encoded, ":", "%3A")

    	graph_sparql = fmt.Sprintf("%s%s", ctx.Configuration.QueryServiceEmbedURL, encoded)
    }

	w.WriteHeader(http.StatusOK)
	t := pongo.Must(pongo.FromFile("templates/article.html"))
	err = t.ExecuteWriter(pongo.Context{
	    "annotations": annotations,
	    "drugs": drugs,
	    "diseases": diseases,
	    "title": title,
	    "article_page_url": article_page_url,
	    "scisource_page_url": scisource_page_url,
	    "wikidata_page_url": wikidata_page_url,
	    "graph_sparql": graph_sparql,
	    "ctx": ctx}, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
