// Shared Utils

package mindbodybrivo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

// Clients and Transports are safe for concurrent use by multiple goroutines and for efficiency should only be created once and re-used
var client = &http.Client{}
var transport = &http.Transport{Proxy: http.ProxyFromEnvironment}

// JSONError is a custom error type for diagnosing server responses
type JSONError struct {
	Code int
	Body map[string]interface{}
}

// Render error message to string
func (e *JSONError) Error() string {
	return fmt.Sprintf("Error code %d and output:\n%+v\n", e.Code, e.Body)
}

// DoRequest is a utility function for making and handling async requests.
// It accepts an http.Request and `output` as pointer to structure that will Unmarshal into.
func DoRequest(req *http.Request, output interface{}) error {
	// Proxy Debugging
	// Enable proxy: export https_proxy="http://localhost:8888"
	if os.Getenv("PROXY") == "true" {
		client = &http.Client{Transport: transport}
	}
	// Disable proxy: unset https_proxy

	// Make request
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Handle response
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	// Check for error response
	if res.StatusCode >= 400 {
		var errorOut map[string]interface{}
		if err = json.Unmarshal(data, &errorOut); err != nil {
			return err
		}
		// Use JSONError type so we can handle specific error codes from the API server
		return &JSONError{res.StatusCode, errorOut}
	}

	// Don't attempt to Unmarshal 204's
	if res.StatusCode == 204 {
		return nil
	}

	// Build response into output *interface{}
	if err = json.Unmarshal(data, output); err != nil {
		return fmt.Errorf("Error unmarshalling json: %s", err)
	}

	// fmt.Printf("Async Output: %+v\n", output)

	return nil
}

// Logger wraps Println in a DEBUG env check
func Logger(message string) {
	if os.Getenv("DEBUG") == "true" {
		fmt.Println(message)
	}
}
