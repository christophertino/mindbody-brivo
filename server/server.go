/**
 * Launch application API to consume MINDBODY webhooks
 *
 * @project 	MINDBODY / Brivo OnAir Membership Sync
 * @author		Christopher Tino
 * @license		MPL 2.0
 */

package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/christophertino/mindbody-brivo/models"
	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
)

// Init : Initialize API routes
func Init(config *models.Config) {
	router := mux.NewRouter()
	// Use wrapper function here so that we can pass `config` to the handler
	router.HandleFunc("/api/v1/user", func(rw http.ResponseWriter, req *http.Request) {
		userHandler(rw, req, config)
	}).Methods("POST")

	// Used by MINDBODY to confirm webhook URL is valid
	router.HandleFunc("/api/v1/user", func(rw http.ResponseWriter, req *http.Request) {
		fmt.Println("Received HEAD Request. Webhook validation successful")
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusNoContent) // Respond with 204
	}).Methods("HEAD")

	// Set default handler
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(rw, "FIAO Brooklyn API")
	})

	server := negroni.New()
	server.UseHandler(router)

	fmt.Printf("Listening for webhook events at PORT %s\n", config.Port)

	http.ListenAndServe(":"+config.Port, server)
}

// Handle request from webhook
func userHandler(rw http.ResponseWriter, req *http.Request, config *models.Config) {
	// Handle request
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Println("server.userHandler: Error reading request", err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// Validate that the request came from MINDBODY
	if !config.Debug {
		if !validateHeader(body, config, req) {
			fmt.Println("server.userHandler: X-Mindbody-Signature is not present or could not be validated")
			rw.WriteHeader(http.StatusForbidden)
			return
		}
	}

	// Build request data into Event model
	var ev models.Event
	if err = json.Unmarshal(body, &ev); err != nil {
		fmt.Println("server.userHandler: Error unmarshalling json", err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// fmt.Printf("UserHandler Output: %+v\n", ev)

	// Respond with 204
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusNoContent)

	// Route event to correct action
	switch ev.EventID {
	case "client.created":
		// Create a new user
		fmt.Println("client.created")
	case "client.updated":
		// Update an existing user
		fmt.Println("client.updated")
	case "client.deactivated":
		// Deactivate an existing user (credential and account)
		fmt.Println("client.deactivated")
	default:
		fmt.Printf("server.userHandler: EventID %s not found\n", ev.EventID)
	}
}

// Check for X-Mindbody-Signature header and validate against encoded request body
func validateHeader(body []byte, config *models.Config, req *http.Request) bool {
	// Remove prepended "sha256=" from header string
	mbSignature := strings.Replace(req.Header.Get("X-Mindbody-Signature"), "sha256=", "", 1)
	if mbSignature != "" {
		// Encode the request body using HMAC-SHA256 and MINDBODY messageSignatureKey
		mac := hmac.New(sha256.New, []byte(config.MindbodyMessageSignatureKey))
		mac.Write(body)
		hash := mac.Sum(nil) // hexidecimal hash

		// Decode the MB header
		decodedHeader, _ := base64.StdEncoding.DecodeString(mbSignature)
		return hmac.Equal(hash, decodedHeader)
	}
	return false
}
