/*

Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.

CODE FROM https://github.com/jgsqware/clairctl

*/
package dockerdist

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

//ErrUnauthorized is return when requested user don't have access to the resource
var ErrUnauthorized = errors.New("unauthorized access")

//bearerAuthParams parse Bearer Token on Www-Authenticate header
func bearerAuthParams(r *http.Response) map[string]string {
	s := strings.SplitN(r.Header.Get("Www-Authenticate"), " ", 2)
	if len(s) != 2 || s[0] != "Bearer" {
		return nil
	}
	result := map[string]string{}

	for _, kv := range strings.Split(s[1], ",") {
		parts := strings.Split(kv, "=")
		if len(parts) != 2 {
			continue
		}
		result[strings.Trim(parts[0], "\" ")] = strings.Trim(parts[1], "\" ")
	}
	return result
}

//AuthenticateResponse add authentication headers on request
func AuthenticateResponse(client *http.Client, dockerResponse *http.Response, request *http.Request) error {
	bearerToken := bearerAuthParams(dockerResponse)
	url := bearerToken["realm"] + "?service=" + url.QueryEscape(bearerToken["service"])
	if bearerToken["scope"] != "" {
		url += "&scope=" + bearerToken["scope"]
	}
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return err
	}

	response, err := client.Do(req)

	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication server response: %v - %v", response.StatusCode, response.Status)
	}

	type token struct {
		Value string `json:"token"`
	}
	var tok token
	err = json.NewDecoder(response.Body).Decode(&tok)

	if err != nil {
		return err
	}

	request.Header.Set("Authorization", "Bearer "+tok.Value)

	return nil
}
