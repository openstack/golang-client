// Copyright (c) 2014 Hewlett-Packard Development Company, L.P.
//
//    Licensed under the Apache License, Version 2.0 (the "License"); you may
//    not use this file except in compliance with the License. You may obtain
//    a copy of the License at
//
//         http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
//    WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
//    License for the specific language governing permissions and limitations
//    under the License.

package misc

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

var zeroByte = &([]byte{}) //pointer to empty []byte

type Session struct {
	Token      string
	TenantName string
	TenantId   string
}

//GetJson sends an Http Request with using the "GET" method and with
//an "Accept" header set to "application/json" and the authenication token
//set to the specified token value. The request is made by the
//specified client. The val interface should be a pointer to the
//structure that the json response should be decoded into.
func GetJson(url string, token string, client http.Client, val interface{}) (err error) {
	req, err := createJsonGetRequest(url, token)
	if err != nil {
		return err
	}

	err = executeRequestCheckStatusDecodeJsonResponse(client, req, val)
	if err != nil {
		return err
	}

	return nil
}

//CallAPI sends an HTTP request using "method" to "url".
//For uploading / sending file, caller needs to set the "content".  Otherwise,
//set it to zero length []byte. If Header fields need to be set, then set it in
// "h".  "h" needs to be even numbered, i.e. pairs of field name and the field
//content.
//
//fileContent, err := ioutil.ReadFile("fileName.ext");
//
//resp, err := CallAPI("PUT", "http://domain/hello/", &fileContent,
//"Name", "world")
//
//is similar to: curl -X PUT -H "Name: world" -T fileName.ext
//http://domain/hello/
func CallAPI(method, url string, content *[]byte, h ...string) (*http.Response, error) {
	if len(h)%2 == 1 { //odd #
		return nil, errors.New("syntax err: # header != # of values")
	}
	//I think the above err check is unnecessary and wastes cpu cycle, since
	//len(h) is not determined at run time. If the coder puts in odd # of args,
	//the integration testing should catch it.
	//But hey, things happen, so I decided to add it anyway, although you can
	//comment it out, if you are confident in your test suites.
	var req *http.Request
	var err error
	req, err = http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(h)-1; i = i + 2 {
		req.Header.Set(h[i], h[i+1])
	}
	req.ContentLength = int64(len(*content))
	if req.ContentLength > 0 {
		req.Body = readCloser{bytes.NewReader(*content)}
		//req.Body = *(new(io.ReadCloser)) //these 3 lines do not work but I am
		//req.Body.Read(content)           //keeping them here in case I wonder why
		//req.Body.Close()                 //I did not implement it this way :)
	}
	return (new(http.Client)).Do(req)
}

type readCloser struct {
	io.Reader
}

func (readCloser) Close() error {
	//cannot put this func inside CallAPI; golang disallow nested func
	return nil
}

//CheckStatusCode compares http response header StatusCode against expected
//statuses. Primary function is to ensure StatusCode is in the 20x (return nil).
//Ok: 200. Created: 201. Accepted: 202. No Content: 204.
//Otherwise return error message.
func CheckHttpResponseStatusCode(resp *http.Response) error {
	switch resp.StatusCode {
	case 200, 201, 202, 204:
		return nil
	case 400:
		return errors.New("Error: response == 400 bad request")
	case 401:
		return errors.New("Error: response == 401 unauthorised")
	case 403:
		return errors.New("Error: response == 403 forbidden")
	case 404:
		return errors.New("Error: response == 404 not found")
	case 405:
		return errors.New("Error: response == 405 method not allowed")
	case 409:
		return errors.New("Error: response == 409 conflict")
	case 413:
		return errors.New("Error: response == 413 over limit")
	case 415:
		return errors.New("Error: response == 415 bad media type")
	case 422:
		return errors.New("Error: response == 422 unprocessable")
	case 429:
		return errors.New("Error: response == 429 too many request")
	case 500:
		return errors.New("Error: response == 500 instance fault / server err")
	case 501:
		return errors.New("Error: response == 501 not implemented")
	case 503:
		return errors.New("Error: response == 503 service unavailable")
	}
	return errors.New("Error: unexpected response status code")
}

func createJsonGetRequest(url string, token string) (req *http.Request, err error) {
	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Auth-Token", token)

	return req, nil
}

func executeRequestCheckStatusDecodeJsonResponse(client http.Client, req *http.Request, val interface{}) (err error) {
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	err = CheckHttpResponseStatusCode(resp)
	if err != nil {
		return err
	}

	err = json.NewDecoder(resp.Body).Decode(&val)
	defer resp.Body.Close()

	return err
}
