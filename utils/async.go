// Copyright 2019 Christopher Tino. All rights reserved.
// This Source Code Form is subject to the terms of the Mozilla Public
// License v. 2.0, which can be found in the LICENSE file.

// Async Utils

package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// JSONError : Custom error type for diagnosing server responses
type JSONError struct {
	Code int
	Body map[string]interface{}
}

func (e *JSONError) Error() string {
	return fmt.Sprintf("async.doRequest: Error code %d and output:\n%+v", e.Code, e.Body)
}

// DoRequest : Utility function for making and handling async requests.
// It accepts an http.Request and `output` as pointer to structure that will Unmarshall into.
func DoRequest(req *http.Request, output interface{}) error {
	// Proxy Debugging
	// var PTransport = &http.Transport{Proxy: http.ProxyFromEnvironment}
	// client := http.Client{Transport: PTransport}

	var client http.Client

	// Make request
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Handle response
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("async.doRequest: Error reading response", err)
		return err
	}

	// Check for error response
	if res.StatusCode >= 400 {
		var errorOut map[string]interface{}
		if err = json.Unmarshal(data, &errorOut); err != nil {
			return err
		}
		return &JSONError{res.StatusCode, errorOut}
	}

	// Don't attempt to unmarshall 204's
	if res.StatusCode == 204 {
		return nil
	}

	// Build response into output *interface{}
	if err = json.Unmarshal(data, output); err != nil {
		fmt.Println("async.doRequest: Error unmarshalling json", err)
		return err
	}

	// fmt.Printf("Async Output: %+v\n", output)

	return nil
}
