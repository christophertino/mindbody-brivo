/**
 * Launch application API to consume MINDBODY webhooks
 *
 * @project 	MINDBODY / Brivo OnAir Membership Sync
 * @author		Christopher Tino
 * @license		MPL 2.0
 */

package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

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
		log.Println("Received HEAD Request. Webhook validation success")
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusNoContent) // Respond with 204
	}).Methods("HEAD")

	router.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(rw, "FIAO Brooklyn API")
	})

	server := negroni.New()
	server.UseHandler(router)

	log.Printf("Listening for webhook events at $PORT %s", config.Port)

	http.ListenAndServe(":"+config.Port, server)
}

// Handle request from webhook
func userHandler(rw http.ResponseWriter, req *http.Request, config *models.Config) {
	// Handle request
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Println("server.userHandler: Error reading request", err)
	}

	// @TODO: validate X-Mindbody-Signature header

	// Build request data into Event model
	var ev models.Event
	if err = json.Unmarshal(body, &ev); err != nil {
		log.Println("server.userHandler: Error unmarshalling json", err)
	}

	// fmt.Printf("UserHandler Output: %+v\n", ev)

	// Respond with 204
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusNoContent)

	// Route event to correct action
	switch ev.EventID {
	case "client.created":
		// Create a new user
		log.Println("client.created")
	case "client.updated":
		// Update an existing user
		log.Println("client.updated")
	case "client.deactivated":
		// Deactivate an existing user (credential and account)
		log.Println("client.deactivated")
	default:
		log.Printf("server.userHandler: EventID %s not found\n", ev.EventID)
	}
}
