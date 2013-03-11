package main

// Copyright ⓒ 2013 Alexander Bauer (see LICENSE.md)

import (
	"encoding/json"
	"net/http"
)

type Summary struct {
	Owner         string    // Owner of the grove instance
	CurrentCommit string    // SHA of the current commit
	Commits       []*Commit // An array of recent commits
	Status        int       // HTTP status
}

func (g *git) ShowJSON(ref string, maxCommits int) (payload string, status int) {
	summary := &Summary{
		Owner:         gitVarUser(),
		CurrentCommit: g.SHA(ref),
		Commits:       g.Commits(ref, maxCommits),
	}
	b, err := json.Marshal(summary)
	if err != nil {
		return "{\"Status\":500}", http.StatusInternalServerError
	}
	return string(b), http.StatusOK
}
