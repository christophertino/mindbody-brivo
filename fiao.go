/**
 * Application Logic
 *
 * Federation of Italian-American Organizations of Brooklyn
 * https://fiaobrooklyn.org/
 *
 * @author		Christopher Tino
 * @license		MPL 2.0
 */

package fiao

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/christophertino/fiao-sync/models"
)

var (
	mb    models.MindBody
	brivo models.Brivo
)

// Authenticate mindbody api
func Authenticate(cj *models.Config) {
	// mbToken := getMindBodyToken(cj)
	// mb.GetClients(cj, mbToken)
	// fmt.Printf("%+v", mb)

	brivoToken := getBrivoToken(cj)
	fmt.Print(brivoToken)
}

func getMindBodyToken(cj *models.Config) string {
	var client http.Client

	// Build request body JSON
	body := map[string]string{
		"Username": cj.MindbodyUsername,
		"Password": cj.MindbodyPassword,
	}
	bytesMessage, err := json.Marshal(body)
	if err != nil {
		log.Fatalln("Error building POST body json", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "https://api.mindbodyonline.com/public/v6/usertoken/issue", bytes.NewBuffer(bytesMessage))
	if err != nil {
		log.Fatalln("Error creating HTTP request", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("SiteId", cj.MindbodySite)
	req.Header.Add("Api-Key", cj.MindbodyAPIKey)

	// Make request
	res, err := client.Do(req)
	if err != nil {
		log.Fatalln("Error fetching MindBody user token", err)
	}
	defer res.Body.Close()

	// Handle response
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalln("Error reading response", err)
	}

	// Build response json
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Fatalln("Error unmarshalling json", err)
	}

	// Look for response errors
	if res.StatusCode >= 400 {
		log.Fatalln("API returned an error", res.StatusCode, result["Error"].(map[string]interface{})["Message"])
	}

	return result["AccessToken"].(string) //cast interface{} to string
}

/**
 * Retrieve a Brivo Access Token using password grant type
 */
func getBrivoToken(cj *models.Config) string {
	var client http.Client

	// Create HTTP request
	req, err := http.NewRequest("POST", fmt.Sprintf("https://auth.brivo.com/oauth/token?grant_type=password&username=%s&password=%s", cj.BrivoUsername, cj.BrivoPassword), nil)
	if err != nil {
		log.Fatalln("Error creating HTTP request", err)
	}
	clientCredentials := base64.StdEncoding.EncodeToString([]byte(cj.BrivoClientID + ":" + cj.BrivoClientSecret))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Basic "+clientCredentials)
	req.Header.Add("api-key", cj.BrivoAPIKey)

	// Make request
	res, err := client.Do(req)
	if err != nil {
		log.Fatalln("Error fetching Brivo access token", err)
	}
	defer res.Body.Close()

	// Handle response
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalln("Error reading response", err)
	}

	// Build response json
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		log.Fatalln("Error unmarshalling json", err)
	}

	// Look for response errors
	if res.StatusCode >= 400 {
		log.Fatalln("API returned an error", res.StatusCode)
	}

	return result["access_token"].(string) //cast interface{} to string
}
