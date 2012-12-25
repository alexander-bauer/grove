package main

import (
	"encoding/json"
	"net/http"
)

type Summary struct {
	Owner         string    //Owner of the grove instance
	CurrentCommit string    //SHA of the current commit
	Commits       []*Commit //An array of recent commits
	Status        int       //HTTP status
}

func ShowJSON(ref, p string, maxCommits int) (payload string, status int) {
	summary := &Summary{
		Owner:         gitVarUser(),
		CurrentCommit: gitSHA(ref, p),
		Commits:       gitCommits(ref, 0, p),
	}
	b, err := json.Marshal(summary)
	if err != nil {
		return "{\"Status\":500}", http.StatusInternalServerError
	}
	return string(b), http.StatusOK
}
