package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAboutHandler(t *testing.T) {

	req, err := http.NewRequest("GET", "/about", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(AboutHandler)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got \n%v want \n%v",
			status, http.StatusOK)
	}

	expected := `{
  "Version": ""
}`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got \n%v want \n%v",
			rr.Body.String(), expected)
	}
}
