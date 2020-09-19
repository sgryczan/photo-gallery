// Scanley
//
// Remote port scanner
//
//   Schemes: https, http
//   BasePath: /
//   Version: 0.0.0
// swagger:meta
package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Version holds the package version
var Version string

type aboutResponse struct {
	Version string
}

// AboutHandler returns information about the api
func AboutHandler(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /about About About
	//
	// Returns information about the application
	// ---
	// consumes:
	// - text/plain
	// produces:
	// - text/plain
	//
	// responses:
	//   '200':
	//     description: About
	//     type: string
	w.WriteHeader(http.StatusOK)
	data := &aboutResponse{
		Version: Version,
	}
	res, _ := json.MarshalIndent(data, "", "  ")

	fmt.Fprintf(w, "%s", res)
}
