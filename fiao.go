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
	"fmt"

	"github.com/christophertino/fiao-sync/models"
)

var (
	auth  models.Auth
	mb    models.MindBody
	brivo models.Brivo
)

// Authenticate mindbody api
func Authenticate(config *models.Config) {
	mbCh := make(chan string)
	brivoCh := make(chan string)

	go auth.MindBodyToken.GetMindBodyToken(config, mbCh)
	go auth.BrivoToken.GetBrivoToken(config, brivoCh)

	fmt.Println(<-mbCh)
	fmt.Println(<-brivoCh)

	fmt.Printf("%+v", auth)

	mb.GetClients(config, auth.MindBodyToken.AccessToken)
	brivo.ListUsers(config, auth.BrivoToken.AccessToken)

	fmt.Printf("%+v", brivo)

	// @TODO:
	// + Map existing user data from MindBody to Brivo
	// + Track matching user IDs using brivo.Data[i].CustomFields
	// + POST new users to Brivo from MindBody
	// + Map user groups
}
