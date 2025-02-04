/* Copyright (c) 2021, VRAI Labs and/or its affiliates. All rights reserved.
 *
 * This software is licensed under the Apache License, Version 2.0 (the
 * "License") as published by the Apache Software Foundation.
 *
 * You may not use this file except in compliance with the License. You may
 * obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
 * License for the specific language governing permissions and limitations
 * under the License.
 */

package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/derekstavis/go-qs"
	"github.com/supertokens/supertokens-golang/recipe/session"
	"github.com/supertokens/supertokens-golang/recipe/thirdparty/tpmodels"
)

func MakeAPIImplementation() tpmodels.APIInterface {
	return tpmodels.APIInterface{
		AuthorisationUrlGET: func(provider tpmodels.TypeProvider, options tpmodels.APIOptions) (tpmodels.AuthorisationUrlGETResponse, error) {
			providerInfo := provider.Get(nil, nil)
			params := map[string]string{}
			for key, value := range providerInfo.AuthorisationRedirect.Params {
				if reflect.ValueOf(value).Kind() == reflect.String {
					params[key] = value.(string)
				} else {
					call, ok := value.(func(req *http.Request) string)
					if ok {
						params[key] = call(options.Req)
					} else {
						return tpmodels.AuthorisationUrlGETResponse{}, errors.New("type of value in params must be a string or a function")
					}
				}
			}
			paramsString, err := getParamString(params)
			if err != nil {
				return tpmodels.AuthorisationUrlGETResponse{}, err
			}
			url := providerInfo.AuthorisationRedirect.URL + "?" + paramsString
			return tpmodels.AuthorisationUrlGETResponse{
				OK: &struct{ Url string }{
					Url: url,
				},
			}, nil
		},

		SignInUpPOST: func(provider tpmodels.TypeProvider, code, redirectURI string, options tpmodels.APIOptions) (tpmodels.SignInUpPOSTResponse, error) {
			providerInfo := provider.Get(&redirectURI, &code)

			accessTokenAPIResponse, err := postRequest(providerInfo)

			if err != nil {
				return tpmodels.SignInUpPOSTResponse{}, err
			}

			userInfo, err := providerInfo.GetProfileInfo(accessTokenAPIResponse)
			if err != nil {
				return tpmodels.SignInUpPOSTResponse{}, err
			}

			emailInfo := userInfo.Email
			if emailInfo == nil {
				return tpmodels.SignInUpPOSTResponse{
					NoEmailGivenByProviderError: &struct{}{},
				}, nil
			}

			response, err := options.RecipeImplementation.SignInUp(provider.ID, userInfo.ID, *emailInfo)
			if err != nil {
				return tpmodels.SignInUpPOSTResponse{}, err
			}
			if response.FieldError != nil {
				return tpmodels.SignInUpPOSTResponse{
					FieldError: &struct{ Error string }{
						Error: response.FieldError.Error,
					},
				}, nil
			}

			if emailInfo.IsVerified {
				tokenResponse, err := options.EmailVerificationRecipeImplementation.CreateEmailVerificationToken(response.OK.User.ID, response.OK.User.Email)
				if err != nil {
					return tpmodels.SignInUpPOSTResponse{}, err
				}
				if tokenResponse.OK != nil {
					_, err := options.EmailVerificationRecipeImplementation.VerifyEmailUsingToken(tokenResponse.OK.Token)
					if err != nil {
						return tpmodels.SignInUpPOSTResponse{}, err
					}
				}
			}

			_, err = session.CreateNewSession(options.Res, response.OK.User.ID, nil, nil)
			if err != nil {
				return tpmodels.SignInUpPOSTResponse{}, err
			}
			return tpmodels.SignInUpPOSTResponse{
				OK: &struct {
					CreatedNewUser   bool
					User             tpmodels.User
					AuthCodeResponse interface{}
				}{
					CreatedNewUser:   response.OK.CreatedNewUser,
					User:             response.OK.User,
					AuthCodeResponse: accessTokenAPIResponse,
				},
			}, nil
		},
	}
}

func postRequest(providerInfo tpmodels.TypeProviderGetResponse) (map[string]interface{}, error) {
	querystring, err := getParamString(providerInfo.AccessTokenAPI.Params)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", providerInfo.AccessTokenAPI.URL, bytes.NewBuffer([]byte(querystring)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("accept", "application/json") // few providers like github don't send back json response by default

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func getParamString(paramsMap map[string]string) (string, error) {
	params := map[string]interface{}{}
	for key, value := range paramsMap {
		params[key] = value
	}
	return qs.Marshal(params)
}
