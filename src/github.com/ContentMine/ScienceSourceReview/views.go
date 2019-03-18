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

type ArticleInfo struct {
	Title  string
	PageID string
	ItemID wikibase.ItemPropertyType
}

/*const ANNOTATION_LIST_QUERY_SPARQL = `
SELECT ?anchor ?annotation ?term ?dictionary ?Wikidata_item_code ?preceding_phrase ?following_phrase ?character_number ?claim WHERE {
  ?anchor wdt:P12 wd:%s.
  ?annotation wdt:P19 ?anchor.
  ?annotation wdt:P15 ?term.
  ?annotation wdt:P16 ?dictionary.
  ?annotation wdt:P2 ?Wikidata_item_code.
  ?anchor wdt:P10 ?character_number.
  OPTIONAL { ?anchor wdt:P13 ?preceding_phrase. }
  OPTIONAL { ?anchor wdt:P14 ?following_phrase. }
  OPTIONAL { ?annotation wdt:P26 ?claim. }
} ORDER BY ?term ASC(?character_number)
`*/
const ANNOTATION_LIST_QUERY_SPARQL = `
SELECT ?anchor ?annotation ?term ?dictionary ?Wikidata_item_code ?preceding_phrase ?following_phrase ?character_number ?claim WHERE {
  ?anchor wdt:P16 wd:%s.
  ?annotation wdt:P21 ?anchor.
  ?annotation wdt:P18 ?term.
  ?annotation wdt:P20 ?dictionary.
  ?annotation wdt:P3 ?Wikidata_item_code.
  ?anchor wdt:P7 ?character_number.
  OPTIONAL { ?anchor wdt:P8 ?preceding_phrase. }
  OPTIONAL { ?anchor wdt:P9 ?following_phrase. }
  OPTIONAL { ?annotation wdt:P22 ?claim. }
} ORDER BY ?term ASC(?character_number)
`

type AnnotationInfo struct {
	AnchorID        wikibase.ItemPropertyType
	AnchorRaw       string
	AnnotationID    wikibase.ItemPropertyType
	AnnotationRaw   string
	Term            string
	Dictionary      string
	WikidataID      string
	PrecedingPhrase string
	FollowingPhrase string
	Offset          string
	Claims          []wikibase.ItemPropertyType
}

type AnnotationSummaryInfo struct {
	WikidataID string
	Dictionary string
	Count      int
}

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

//const ENTITY_PREFIX = "http://sciencesource.wmflabs.org/entity/"
const ENTITY_PREFIX = "http://wikibase.svc/entity/"


type ClaimInfo struct {
    Drug *AnnotationInfo
    Disease *AnnotationInfo
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
			ItemID: wikibase.ItemPropertyType(strings.TrimPrefix(binding["res"].Value, ENTITY_PREFIX)),
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

func getArticleAnnotationList(queryservice_url string, article_id string) ([]*AnnotationInfo, map[string]AnnotationSummaryInfo, error) {

	resp, err := wikibase.MakeSPARQLQuery(queryservice_url, fmt.Sprintf(ANNOTATION_LIST_QUERY_SPARQL, article_id))
	if err != nil {
		return nil, nil, err
	}

	annotations := make([]*AnnotationInfo, 0, len(resp.Results.Bindings))
	summaries := make(map[string]AnnotationSummaryInfo, 0)

    var previous_annotation *AnnotationInfo
	for _, binding := range resp.Results.Bindings {

		term := binding["term"].Value
		anchor_id := wikibase.ItemPropertyType(strings.TrimPrefix(binding["anchor"].Value, ENTITY_PREFIX))
		annotation_id := wikibase.ItemPropertyType(strings.TrimPrefix(binding["annotation"].Value, ENTITY_PREFIX))

        var annotation *AnnotationInfo
        if previous_annotation != nil && previous_annotation.AnchorID == anchor_id {
            annotation = previous_annotation
        } else {
            new_annotation := AnnotationInfo{
                AnchorID:        anchor_id,
                AnchorRaw:       binding["anchor"].Value,
                AnnotationID:    annotation_id,
                AnnotationRaw:   binding["annotation"].Value,
                Term:            binding["term"].Value,
                Dictionary:      binding["dictionary"].Value,
                WikidataID:      binding["Wikidata_item_code"].Value,
                Offset:          binding["character_number"].Value,
                PrecedingPhrase: binding["preceding_phrase"].Value,
                FollowingPhrase: binding["following_phrase"].Value,
                Claims:          make([]wikibase.ItemPropertyType, 0),
            }
            annotation = &new_annotation
    		annotations = append(annotations, annotation)

            // only update summary when we have a new term
    		summary, ok := summaries[term]
            if ok {
                summary.Count += 1
            } else {
                summary = AnnotationSummaryInfo{
                    WikidataID: binding["Wikidata_item_code"].Value,
                    Dictionary: binding["dictionary"].Value,
                    Count:      1,
                }
            }

            summaries[term] = summary
        }
		claim := binding["claim"].Value
        if claim != "" {
            annotation.Claims = append(annotation.Claims, wikibase.ItemPropertyType(strings.TrimPrefix(claim, ENTITY_PREFIX)))
        }
		previous_annotation = annotation


	}

	return annotations, summaries, nil
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

	annotations, summaries, err := getArticleAnnotationList(ctx.Configuration.QueryServiceURL, id)
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
	drugs := make([]*AnnotationInfo, 0)
	diseases := make([]*AnnotationInfo, 0)
	for _, annotation := range annotations {
		dict := annotation.Dictionary
		if strings.Contains(dict, "drug") {
			drug_dictionary = dict
			drugs = append(drugs, annotation)
		} else {
			disease_dictionary = dict
			diseases = append(diseases, annotation)
		}
	}

	if (disease_dictionary != "") && (drug_dictionary != "") {
		encoded := url.PathEscape(fmt.Sprintf(GRAPH_SPARQL, id, id, disease_dictionary, drug_dictionary))
		encoded = strings.ReplaceAll(encoded, ":", "%3A")

		graph_sparql = fmt.Sprintf("%s%s", ctx.Configuration.QueryServiceEmbedURL, encoded)
	}

	// Generate a nice lookup set for checking viewews
	set := make(map[wikibase.ItemPropertyType]*AnnotationInfo, 0)
	for _, annotation := range annotations {
        set[annotation.AnnotationID] = annotation
	}
    claims := make([]ClaimInfo, 0)
    for _, annotation := range annotations {
        for _, claim := range annotation.Claims {
            new_claim := ClaimInfo{
                Drug: annotation,
                Disease: set[claim],
            }
            claims = append(claims, new_claim)
        }
    }

	w.WriteHeader(http.StatusOK)
	t := pongo.Must(pongo.FromFile("templates/article.html"))
	err = t.ExecuteWriter(pongo.Context{
		"summaries":          summaries,
		"drugs":              drugs,
		"diseases":           diseases,
		"title":              title,
		"claims": claims,
		"article_page_url":   article_page_url,
		"scisource_page_url": scisource_page_url,
		"wikidata_page_url":  wikidata_page_url,
		"graph_sparql":       graph_sparql,
		"ctx":                ctx}, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func recordClaim(ctx *ServerContext, drug_annotation *AnnotationInfo, disease_annotation *AnnotationInfo) error {

	access_token := wikibase.AccessToken{
		Token:  ctx.AccessToken.Token,
		Secret: ctx.AccessToken.Secret,
	}
	oauth_info := wikibase.OAuthInformation{
		Consumer: ctx.Configuration.OAuthConsumer,
		Access:   &access_token,
	}
	oauth_client := wikibase.NewOAuthNetworkClient(oauth_info, ctx.Configuration.WikibaseURL)
	wikibase_client := wikibase.NewClient(oauth_client)

	// We will need an editing token
	_, err := wikibase_client.GetEditingToken()
	if err != nil {
		return err
	}

	item_claim, err := wikibase.ItemClaimToAPIData(disease_annotation.AnnotationID)
	if err != nil {
		return err
	}
	item_data, err := json.Marshal(item_claim)
	if err != nil {
		return err
	}
	_, err = wikibase_client.CreateClaimOnItem(drug_annotation.AnnotationID, CLAIM_PROPERTY_ID, item_data)

	return err
}

func reviewHandler(ctx *ServerContext, w http.ResponseWriter, r *http.Request) {

	// Should only be called by POST
	if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		log.Printf("Error parsing form: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	drug_id := wikibase.ItemPropertyType(r.FormValue("drug"))
	disease_id := wikibase.ItemPropertyType(r.FormValue("disease"))
	confirm := r.FormValue("confirm")

	vars := mux.Vars(r)
	id := vars["id"]

	properties, err := getArticleProperties(ctx.Configuration.QueryServiceURL, id)
	if err != nil {
		log.Printf("Error making property query: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	title := properties[TITLE_PROPERTY]

	annotations, _, err := getArticleAnnotationList(ctx.Configuration.QueryServiceURL, id)
	if err != nil {
		log.Printf("Error making annotation query: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var drug_annotation *AnnotationInfo
	var disease_annotation *AnnotationInfo
	for _, annotation := range annotations {
		if annotation.AnnotationID == drug_id {
			drug_annotation = annotation
		}
		if annotation.AnnotationID == disease_id {
			disease_annotation = annotation
		}
	}

	if drug_annotation == nil || disease_annotation == nil {
		log.Printf("We have missing annotation info: %v %v", drug_id, disease_id)
		http.Error(w, "Form data missing", http.StatusBadRequest)
		return
	}

	if confirm == "true" {
		err := recordClaim(ctx, drug_annotation, disease_annotation)
		if err != nil {
			log.Printf("Failed to record claim: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "../", http.StatusTemporaryRedirect)
		return
	}

	w.WriteHeader(http.StatusOK)
	t := pongo.Must(pongo.FromFile("templates/review.html"))
	err = t.ExecuteWriter(pongo.Context{
		"title":   title,
		"drug":    drug_annotation,
		"disease": disease_annotation,
		"ctx":     ctx,
	}, w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
