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
func DoRequest(req *http.Request, output interface{}) (interface{}, error) {
	var client http.Client

	// Make request
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Handle response
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("async.doRequest: Error reading response", err)
		return nil, err
	}

	// Check for error response
	if res.StatusCode >= 400 {
		var errorOut map[string]interface{}
		if err = json.Unmarshal(data, &errorOut); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("async.doRequest: Error creating credential \n %+v", errorOut)
	}

	// Build response into Model
	if err = json.Unmarshal(data, &output); err != nil {
		log.Println("async.doRequest: Error unmarshalling json", err)
		return nil, err
	}

	// fmt.Printf("%+v", output)

	return output, nil
}
