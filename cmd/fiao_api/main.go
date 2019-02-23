/**
 * Application Init
 *
 * Federation of Italian-American Organizations of Brooklyn
 * https://fiaobrooklyn.org/
 *
 * @author		Christopher Tino
 * @license		MPL 2.0
 */

package main

import (
	fiaoapi "github.com/christophertino/fiao_api"
)

func main() {
	fiaoapi.Authenticate()
}
