package handlers

import "net/http"

// HomeHandler redirects to swaggerui
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/api/", 302)
}
