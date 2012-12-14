package main

import (
	"io/ioutil"
	"os"
	"path"
	"strings"
)

//ShowPath takes a fully rooted path as an argument, and generates an HTML webpage in order in order to allow the user to navigate or clone via http. It expects the given URL to have a trailing "/".
func ShowPath(url string, p string) (page string) {
	css, err := ioutil.ReadFile("style.css")
	if err != nil {
		panic(err)
	}

	//Retrieve information about the file.
	fi, err := os.Stat(p)
	if err != nil {
		//If there is an error, present
		//a 404.
		//TODO create an actual error
		return "404"
	}
	if !fi.IsDir() {
		//If the file is not a directory,
		//then return 403 unauthorized.
		//TODO create an actual error
		return "403"
	}

	f, err := os.Open(p)
	if err != nil || f == nil {
		//If there is an error opening
		//the file, return 500.
		//TODO
		return "500"
	}
	names, err := f.Readdirnames(0)
	f.Close()
	if err != nil {
		//If the directory could not be
		//opened, return 500.
		return "500"
	}

	//Find whether the directory contains
	//a .git file.
	//TODO find if the directory is a
	//bare git repository (name.git)
	var isGit bool
	var gitDir string
	for _, name := range names {
		if name == ".git" {
			isGit = true
			gitDir = name
			break
		}
	}

	if isGit {
		html := "<html><head><style type=\"text/css\">" + string(css) + "</style></head><body><div class=\"title\"><a href=\"" + url + "..\">.. / </a>" + path.Base(p) + "</div>"
		//now add the button things
		html += "<div class=\"wrapper\"><div class=\"button\"><div class=\"buttontitle\">Current Branch</div><br/><div class=\"buttontext\">Master</div></div><div class=\"button\"><div class=\"buttontitle\">Branches</div><br/><div class=\"buttontext\">3</div></div><div class=\"button\"><div class=\"buttontitle\">Commits</div><br/><div class=\"buttontext\">3</div></div><div class=\"button\"><div class=\"buttontitle\">Current Commit</div><br/><div class=\"buttontext\">503099ca5b</div></div></div>"
		//now everything else for right now
		html += url + gitDir + "</body></html>"

		return html
	} else {
		var dirList string = "<ul>"
		if url != "/" {
			dirList += "<a href=\"" + url + "..\"><li>..</li></a>"
		}
		for _, name := range names {
			if !strings.HasPrefix(name, ".") {
				dirList += "<a href=\"" + url + name + "\"><li>" + name + "</li></a>"
			}
		}
		page = "<html><head><style type=\"text/css\">" + string(css) + "</style></head><body>Welcome to <a href=\"https://github.com/SashaCrofter/grove\">grove</a>.<br/>" + dirList + "</ul></body></html>"
	}
	return
}
