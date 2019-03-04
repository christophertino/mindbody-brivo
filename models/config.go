/**
 * Configuration Data Model
 *
 * Federation of Italian-American Organizations of Brooklyn
 * https://fiaobrooklyn.org/
 *
 * @author		Christopher Tino
 * @license		MPL 2.0
 */

package models

// Config : Settings imported from conf.json
type Config struct {
	BrivoUsername     string `json:"brivo_username"`
	BrivoPassword     string `json:"brivo_password"`
	BrivoClientID     string `json:"brivo_client_id"`
	BrivoClientSecret string `json:"brivo_client_secret"`
	BrivoAPIKey       string `json:"brivo_api_key"`

	MindbodyAPIKey   string `json:"mindbody_api_key"`
	MindbodyUsername string `json:"mindbody_username"`
	MindbodyPassword string `json:"mindbody_password"`
	MindbodySite     string `json:"mindbody_site"`
}
