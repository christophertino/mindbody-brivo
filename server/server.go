// Launch application API to consume MINDBODY webhooks

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

	db "github.com/christophertino/mindbody-brivo"
	utils "github.com/christophertino/mindbody-brivo"
	"github.com/christophertino/mindbody-brivo/models"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
)

var (
	auth         models.Auth
	pool         *redis.Pool
	isRefreshing bool
	errChan      chan *models.Event
)

// Launch will start the web server and initialize API routes
func Launch(config *models.Config) {
	router := mux.NewRouter()

	// Create new Redis connection pool
	pool = db.NewPool(config.RedisURL)

	// Handle MINDBODY webhook events for client updates
	router.HandleFunc("/api/v1/user", func(rw http.ResponseWriter, req *http.Request) {
		// Use wrapper function here so that we can pass `config` to the handler
		userHandler(rw, req, config)
	}).Methods(http.MethodPost)

	// Handle Brivo event subscriptions for site access
	router.HandleFunc("/api/v1/access", func(rw http.ResponseWriter, req *http.Request) {
		accessHandler(rw, req, config)
	}).Methods(http.MethodPost)

	// Used by MINDBODY to confirm webhook URL is valid
	router.HandleFunc("/api/v1/user", func(rw http.ResponseWriter, req *http.Request) {
		fmt.Println("Received HEAD Request. Webhook validation successful")
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusAccepted) // Respond with 202
	}).Methods(http.MethodHead)

	// Set default handler
	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(rw, "Mindbody-Brivo API")
	})

	server := negroni.New()
	server.UseHandler(router)

	// Generate access tokens for Brivo and Mindbody
	if err := auth.Authenticate(config); err != nil {
		log.Fatalf("Error generating access tokens: %s", err)
	}

	// Create buffer channel to handle errors from the userHandler
	errChan = make(chan *models.Event, config.BrivoRateLimit)
	isRefreshing = false

	fmt.Printf("Listening for events at PORT %s\n", config.Port)

	http.ListenAndServe(":"+config.Port, server)
}

// Handle MINDBODY webhook requests
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

	// Debug webhook payload
	utils.Logger(fmt.Sprintf("EventData payload:\n%+v", event.EventData))

	// Check current refresh status
	if !isRefreshing {
		// Process the event normally
		go event.ProcessEvent(errChan, isRefreshing, config, &auth)
	} else {
		// A refresh is currently taking place. Push the event into the error channel
		errChan <- &event
	}
}

// Handle Brivo access requests
func accessHandler(rw http.ResponseWriter, req *http.Request, config *models.Config) {
	// Handle request
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Println("Error reading request", err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// Build request data into Access model
	var access models.Access
	if err = json.Unmarshal(body, &access); err != nil {
		fmt.Println("Error unmarshalling json", err)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// Respond with 200 (Brivo doesn't like 202)
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)

	// Debug webhook payload
	utils.Logger(fmt.Sprintf("Access data payload:\n%+v", access))

	// Process the access request
	access.ProcessRequest(config, &auth, pool)
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

		// Decode the MINDBODY header
		decodedHeader, _ := base64.StdEncoding.DecodeString(mbSignature)
		return hmac.Equal(hash, decodedHeader)
	}
	return false
}
