//
// Copyright (c) 2020 SSH Communications Security Inc.
//
// All rights reserved.
//

package restapi_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/SSHcom/privx-sdk-go/restapi"
)

//
type mockIdP struct{ headers map[string]string }

func (idp mockIdP) Headers() (map[string]string, error) {
	return idp.headers, nil
}

var access = restapi.AccessToken(
	mockIdP{map[string]string{
		"Authorization": "Bearer trusted",
	}},
)

func TestGet(t *testing.T) {
	ts := mockStatus()
	defer ts.Close()

	_, err := restapi.New(restapi.Endpoint(ts.URL)).
		Get("/users/%v", 1).RecvStatus()

	if err != nil {
		t.Errorf("client get fails: %w", err)
	}
}

func TestGetFails(t *testing.T) {
	ts := mockStatus()
	defer ts.Close()

	err := restapi.New(restapi.Endpoint(ts.URL)).
		Get("/users/%v", 2).RecvStatus()

	if err == nil {
		t.Errorf("client get is not failing.")
	} else if err.Error() !=
		"error: error42, message: borken request, property: mock" {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestSend(t *testing.T) {
	ts := mockStatus()
	defer ts.Close()

	type T struct {
		ID string `json:"id"`
	}

	methods := []restapi.CURL{
		restapi.New(restapi.Endpoint(ts.URL)).Put("/echo"),
		restapi.New(restapi.Endpoint(ts.URL)).Post("/echo"),
	}

	for _, method := range methods {
		eg := T{ID: "id"}
		in := T{}

		err := method.Send(eg).Recv(&in)
		if err != nil {
			t.Errorf("client fails: %w", err)
		}

		if eg.ID != in.ID {
			t.Errorf("unexpected response: %v", in)
		}
	}
}

func TestRecv(t *testing.T) {
	ts := mock()
	defer ts.Close()

	var data struct {
		ID string `json:"id"`
	}

	err := restapi.New(access, restapi.Endpoint(ts.URL)).
		Get("/").
		Recv(&data)

	if err != nil {
		t.Errorf("client fails: %w", err)
	}

	if data.ID != "trusted" {
		t.Errorf("unexpected response: %v", data)
	}
}

func TestRecvNoIdP(t *testing.T) {
	ts := mock()
	defer ts.Close()

	var data struct {
		ID string `json:"id"`
	}

	err := restapi.New(restapi.Endpoint(ts.URL)).
		Get("/").
		Recv(&data)

	if err != nil {
		t.Errorf("client fails: %w", err)
	}

	if data.ID != "untrusted" {
		t.Errorf("unexpected response: %v", data)
	}
}

//
func mock() *httptest.Server {
	return httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Header.Get("Authorization") == "Bearer trusted":
				w.Header().Add("Content-Type", "application/json")
				w.Write([]byte(`{"id": "trusted"}`))
			default:
				w.Header().Add("Content-Type", "application/json")
				w.Write([]byte(`{"id": "untrusted"}`))
			}
		}),
	)
}

//
func mockStatus() *httptest.Server {
	return httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/users/1":
				w.WriteHeader(http.StatusOK)
			case strings.HasPrefix(r.URL.Path, "/users/"):
				w.WriteHeader(http.StatusBadRequest)
				body, _ := json.Marshal(map[string]string{
					"error_code":    "error42",
					"error_message": "borken request",
					"property":      "mock",
				})
				w.Write(body)

			case r.URL.Path == "/echo":
				b, _ := ioutil.ReadAll(r.Body)
				w.Write(b)
			}
		}),
	)
}
