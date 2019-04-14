/**
 * HTTP Server
 *
 * Federation of Italian-American Organizations of Brooklyn
 * https://fiaobrooklyn.org/
 *
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

	"github.com/christophertino/fiao-sync/models"
	"github.com/gorilla/mux"
	"github.com/urfave/negroni"
)

// Init : Initialize API routes
func Init() {
	router := mux.NewRouter()
	router.HandleFunc("/api/user", userHandler).Methods("POST")

	server := negroni.New()
	server.UseHandler(router)

	fmt.Println("Listening for webhook events at localhost:3002")

	http.ListenAndServe("localhost:3002", server)
}

// Handle request from webhook
func userHandler(rw http.ResponseWriter, req *http.Request) {
	// Handle request
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Println("server.userHandler: Error reading request", err)
	}

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
	case "client.updated":
		// Update an exisitng user
	case "client.deactivated":
		// Deactivate an existing user (credential and account)
	default:
		log.Printf("server.userHandler: EventID %s not found\n", ev.EventID)
	}
}
