// Copyright (c) 2017, Oracle and/or its affiliates. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
)

// OCIResponseStub stubs out http requests and returns a response if found in the
// map provided as input. If the response for the stubbed out http request had not been
// provided, a 404 error is retured
func OCIResponseStub(expectedResponseMap map[string]*string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedResponse, err := expectedResponseMap[r.RequestURI]
		fmt.Printf("Expected response %v", err)
		if err != true {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Write([]byte(*expectedResponse))
	}))
}
