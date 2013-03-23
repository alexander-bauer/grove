package main

// Copyright ⓒ 2013 Alexander Bauer and Luke Evers (see LICENSE.md)

import (
	"encoding/json"
	"errors"
	"net/http"
)

// encoder is a private interface which all encoding/*.Encoder types
// implement. It is used in ServeAPI.
type encoder interface {
	Encode(v interface{}) error
}

type APIResponse struct {
	GroveOwner string    // Owner of the grove instance
	Ref        string    // Current git ref or branch name
	Commits    []*Commit // Commits in which the most recent is first
	Error      string    `json:",omitempty"` // Error string if present
}

var (
	InvalidEncodingError = errors.New("api: invalid encoding requested")
)

func ServeAPI(w http.ResponseWriter, req *http.Request, g *git, ref string, maxCommits int) (err error) {
	// First, determine the encoding and error if it isn't appropriate
	// or supported.
	var e encoder
	switch req.FormValue("api") {
	case "json":
		// The json.Encoder type implements our private encoder
		// interface, because it has the function Encode().
		e = json.NewEncoder(w)
	}
	if e == nil {
		// If the encoding is invalid or not provided, return this
		// error.
		return InvalidEncodingError
	}
	// If an encoding was provided, prepare a response.
	r := &APIResponse{
		GroveOwner: gitVarUser(),
		Ref:        g.SHA("HEAD"),
		Commits:    g.Commits(ref, maxCommits),
	}

	// Finally, encode to the http.ResponseWriter with whatever
	// encoder was selected.
	return e.Encode(r)
}
