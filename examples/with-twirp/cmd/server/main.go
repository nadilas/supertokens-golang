// Copyright 2018 Twitch Interactive, Inc.  All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not
// use this file except in compliance with the License. A copy of the License is
// located at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// or in the "license" file accompanying this file. This file is distributed on
// an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/supertokens/supertokens-golang/examples/with-twirp/haberdasher"
	"github.com/supertokens/supertokens-golang/examples/with-twirp/internal/haberdasherserver"
	"github.com/supertokens/supertokens-golang/examples/with-twirp/internal/hooks"
	"github.com/supertokens/supertokens-golang/examples/with-twirp/internal/interceptor"
	"github.com/supertokens/supertokens-golang/recipe/emailpassword"
	"github.com/supertokens/supertokens-golang/recipe/session"
	"github.com/supertokens/supertokens-golang/recipe/session/sessmodels"
	"github.com/supertokens/supertokens-golang/supertokens"
	"github.com/twitchtv/twirp"
)

func main() {
	err := supertokens.Init(supertokens.TypeInput{
		Supertokens: &supertokens.ConnectionInfo{
			ConnectionURI: "https://try.supertokens.io",
		},
		AppInfo: supertokens.AppInfo{
			AppName:       "SuperTokens Demo App",
			APIDomain:     "http://localhost:3001",
			WebsiteDomain: "http://localhost:3000",
		},
		RecipeList: []supertokens.Recipe{
			emailpassword.Init(nil),
			session.Init(nil),
		},
	})

	if err != nil {
		panic(err.Error())
	}

	hook := hooks.LoggingHooks(os.Stderr)
	service := haberdasherserver.New()
	server := haberdasher.NewHaberdasherServer(service,
		twirp.WithServerInterceptors(interceptor.NewSuperTokensErrorHandlerInterceptor()), hook)
	sessionRequired := false
	log.Fatal(http.ListenAndServe(":3001", handlers.CORS(
		handlers.AllowedHeaders(append([]string{"Content-Type"}, supertokens.GetAllCORSHeaders()...)),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS"}),
		handlers.AllowedOrigins([]string{"http://localhost:3000"}),
		handlers.AllowCredentials(),
	)(supertokens.Middleware(session.VerifySession(&sessmodels.VerifySessionOptions{
		SessionRequired: &sessionRequired,
	}, server.ServeHTTP)))))
}
