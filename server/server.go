// Launch application API to consume MINDBODY webhooks
//
// Copyright 2019 Christopher Tino. All rights reserved.

package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	utils "github.com/christophertino/mindbody-brivo"
	"github.com/christophertino/mindbody-brivo/models"
	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
)

var (
	auth         models.Auth
	mu           sync.Mutex
	isRefreshing bool
	errChan      chan *models.Event
)

// Launch will start the web server and initialize API routes
func Launch(config *models.Config) {
	router := mux.NewRouter()

	// Handle webhook events related to MINDBODY clients
	router.HandleFunc("/api/v1/user", func(rw http.ResponseWriter, req *http.Request) {
		// Use wrapper function here so that we can pass `config` to the handler
		userHandler(rw, req, config)
	}).Methods(http.MethodPost)

	// Used by MINDBODY to confirm webhook URL is valid
	router.HandleFunc("/api/v1/user", func(rw http.ResponseWriter, req *http.Request) {
		fmt.Println("Received HEAD Request. Webhook validation successful")
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusAccepted) // Respond with 202
	}).Methods(http.MethodHead)

	// Set default handler
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(rw, "FIAO Brooklyn API")
	})

	server := negroni.New()
	server.UseHandler(router)

	// Generate Brivo access token. MINDBODY token not needed
	if err := auth.BrivoToken.GetBrivoToken(config); err != nil {
		log.Fatalf("Error generating Brivo access token: %s", err)
	}

	// Create buffer channel to handle errors from the userHandler
	errChan = make(chan *models.Event, config.BrivoRateLimit)
	isRefreshing = false

	fmt.Printf("Listening for webhook events at PORT %s\n", config.Port)

	http.ListenAndServe(":"+config.Port, server)
}

// Handle request from webhook
func userHandler(rw http.ResponseWriter, req *http.Request, config *models.Config) {
	// Handle request
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Println("Error reading request", err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// Validate that the request came from MINDBODY
	if !config.Debug {
		if !validateHeader(body, *config, req) {
			fmt.Println("X-Mindbody-Signature is not present or could not be validated")
			rw.WriteHeader(http.StatusForbidden)
			return
		}
	}

	// Build request data into Event model
	var event models.Event
	if err = json.Unmarshal(body, &event); err != nil {
		fmt.Println("Error unmarshalling json", err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// Respond with 202
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusAccepted)

	// Validate that the ClientID is a valid hex ID
	if !models.IsValidID(event.EventData.ClientID) {
		utils.Logger(fmt.Sprintf("User %s is not a valid hex ID", event.EventData.ClientID))
		return
	}

	// Debug webhook payload
	utils.Logger(fmt.Sprintf("EventData payload:\n%+v", event.EventData))

	// Check current refresh status
	if !isRefreshing {
		// Process the event normally
		go processEvent(&event, config)
	} else {
		// A refresh is currently taking place. Push the event into the error channel
		errChan <- &event
	}
}

// Handle cases for each webhook EventID
func processEvent(event *models.Event, config *models.Config) {
	// Route event to correct action
	switch event.EventID {
	case "client.created":
		// Create a new user
		fallthrough
	case "client.updated":
		// Update an existing user
		if err := event.CreateOrUpdateUser(*config, auth); err != nil {
			// If we get a 401:Unauthorized, the token is expired
			if err.Error() == "401" {
				// Stash the current event in the error channel
				errChan <- event
				// Handle token refresh
				doRefresh(config)
				break
			}
			fmt.Printf("Error creating/updating Brivo client with MINDBODY ID %d\n%s", event.EventData.ClientUniqueID, err)
		}
	case "client.deactivated":
		// Suspend an existing user
		if err := event.DeactivateUser(*config, auth); err != nil {
			// If we get a 401:Unauthorized, the token is expired
			if err.Error() == "401" {
				// Stash the current event in the error channel
				errChan <- event
				// Handle token refresh
				doRefresh(config)
				break
			}
			fmt.Printf("Error deactivating Brivo client with MINDBODY ID %d\n%s", event.EventData.ClientUniqueID, err)
		}
	default:
		fmt.Printf("EventID %s not found\n", event.EventID)
	}
}

// Check current refreshing status and process new refresh token
func doRefresh(config *models.Config) {
	if isRefreshing {
		return
	}

	// Lock the refresh sequence as there may be multiple routines attempting to refresh at once
	mu.Lock()
	isRefreshing = true

	// Check that token hasn't already been refreshed
	if time.Now().UTC().After(auth.BrivoToken.ExpireTime) {
		if err := auth.BrivoToken.RefreshBrivoToken(*config); err != nil {
			fmt.Println("Error refreshing Brivo AUTH token:\n", err)
			return
		}
		utils.Logger("Refreshed Brivo AUTH token")
	}

	isRefreshing = false
	mu.Unlock()

	// Listen for new events in the error channel
loop:
	for {
		select {
		case event := <-errChan:
			go processEvent(event, config)
		default:
			break loop
		}
	}
}

// Check for X-Mindbody-Signature header and validate against encoded request body
func validateHeader(body []byte, config models.Config, req *http.Request) bool {
	// Remove prepended "sha256=" from header string
	mbSignature := strings.Replace(req.Header.Get("X-Mindbody-Signature"), "sha256=", "", 1)
	if mbSignature != "" {
		// Encode the request body using HMAC-SHA256 and MINDBODY messageSignatureKey
		mac := hmac.New(sha256.New, []byte(config.MindbodyMessageSignatureKey))
		mac.Write(body)
		hash := mac.Sum(nil) // hexadecimal hash

		// Decode the MB header
		decodedHeader, _ := base64.StdEncoding.DecodeString(mbSignature)
		return hmac.Equal(hash, decodedHeader)
	}
	return false
}
