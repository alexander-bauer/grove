package main

// Copyright ⓒ 2013 Alexander Bauer and Luke Evers (see LICENSE.md)

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"net/http"
	"strings"
)

// encoder is a private interface which all encoding/*.Encoder types
// implement. It is used in ServeAPI.
type encoder interface {
	Encode(v interface{}) error
}

type APIResponse struct {
	GroveOwner  string    // Owner of the grove instance
	HEAD        string    // The current ref of HEAD
	Description string    // Current branch description if available
	Commits     []*Commit // Commits in which the most recent is first
	Error       string    `json:",omitempty"` // Error string if present
}

var (
	InvalidEncodingError = errors.New("api: invalid encoding requested")
)

func ServeAPI(w http.ResponseWriter, req *http.Request, g *git, ref string, maxCommits int) (err error) {
	// First, determine the encoding and error if it isn't appropriate
	// or supported. To do this, we need to check the api value and
	// Accept header. We also want to include the Content-Type.
	var e encoder
	var c string // The Content-Type field in the http.Response
	switch req.FormValue("api") {
	case "json":
		// The json.Encoder type implements our private encoder
		// interface, because it has the function Encode().
		c = "application/json"
		e = json.NewEncoder(w)
	case "xml":
		// Same as above.
		c = "application/xml"
		e = xml.NewEncoder(w)
	}
	// If the api field wasn't submitted in the form, we should still
	// check the Accept header.
	accept := strings.Split(strings.Split(
		req.Header.Get("Accept"), ";")[0],
		",")
	if e == nil && len(accept) != 0 {
		// Now we must loop through each element in accept, because
		// there can be multiple values to the Accept key. As soon as
		// we find an acceptable encoding, break the loop.
		for _, a := range accept {
			switch {
			case strings.HasSuffix(a, "/json"):
				e = json.NewEncoder(w)
			case strings.HasSuffix(a, "/xml"):
				e = xml.NewEncoder(w)
			}
			if e != nil {
				c = a // Set the content type appropriately.
				break
			}
		}
	}
	// If the encoding is invalid or not provided, return this error.
	if e == nil {
		return InvalidEncodingError
	}

	// If an encoding was provided, prepare a response.
	r := &APIResponse{
		GroveOwner:  user,
		HEAD:        g.SHA("HEAD"),
		Description: g.GetBranchDescription(ref),
		Commits:     g.Commits(ref, maxCommits),
	}
	// Set the Content-Type appropriately in the header.
	w.Header().Set("Content-Type", c)

	// Finally, encode to the http.ResponseWriter with whatever
	// encoder was selected.
	return e.Encode(r)
}
