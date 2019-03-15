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
	"log"
	"net/http"

	"github.com/mrjones/oauth"
)

func authHandler(ctx *ServerContext, w http.ResponseWriter, r *http.Request) {

	ctx.OAuthConsumer.AdditionalParams = map[string]string{
		"title": "Special:OAuth/initiate",
	}

	token, requestUrl, err := ctx.OAuthConsumer.GetRequestTokenAndUrl("oob")
	if err != nil {
		log.Printf("Error getting token: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Make sure to save the token, we'll need it for AuthorizeToken()
	ctx.CookieSession.Values[token.Token] = token.Secret
	ctx.CookieSession.Save(r, w)

	http.Redirect(w, r, requestUrl, http.StatusTemporaryRedirect)
}

func getTokenHandler(ctx *ServerContext, w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	verificationCode := values.Get("oauth_verifier")
	tokenKey := values.Get("oauth_token")

	val := ctx.CookieSession.Values[tokenKey]
	if secret, ok := val.(string); ok {
		request := oauth.RequestToken{Token: tokenKey, Secret: secret}
		ctx.OAuthConsumer.AdditionalParams = map[string]string{
			"title": "Special:OAuth/token",
		}
		accessToken, err := ctx.OAuthConsumer.AuthorizeToken(&request, verificationCode)
		if err != nil {
			log.Printf("Error getting access token: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ctx.CookieSession.Values["auth"] = &accessToken
		ctx.CookieSession.Save(r, w)

		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	} else {
		log.Printf("Failed to get request data")
		http.Error(w, "Failed to get request data", http.StatusInternalServerError)
		return
	}
}

func deauthHandler(ctx *ServerContext, w http.ResponseWriter, r *http.Request) {

	// Setting the max age to -ve should delete the cookie session
	ctx.CookieSession.Options.MaxAge = -1
	ctx.CookieSession.Save(r, w)

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
