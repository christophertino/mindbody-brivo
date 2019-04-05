/**
 * Async Utils
 *
 * Federation of Italian-American Organizations of Brooklyn
 * https://fiaobrooklyn.org/
 *
 * @author		Christopher Tino
 * @license		MPL 2.0
 */

package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// DoRequest : Utility function for making and handling async requests
// @param	http.Request
// @param	output		pointer to structure that we will Unmarshall into
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
		log.Println("async.doRequest: Error reading response", err)
		return err
	}

	// Check for error response
	if res.StatusCode >= 400 {
		var errorOut map[string]interface{}
		if err = json.Unmarshal(data, &errorOut); err != nil {
			return err
		}
		return fmt.Errorf("async.doRequest: Error code %d and output: \n %+v", res.StatusCode, errorOut)
	}

	// Don't attempt to unmarshall 204's
	if res.StatusCode == 204 {
		return nil
	}

	// Build response into output *interface{}
	if err = json.Unmarshal(data, output); err != nil {
		log.Println("async.doRequest: Error unmarshalling json", err)
		return err
	}

	// fmt.Printf("%+v \n", output)

	return nil
}
