package main

import (
	"io/ioutil"
	"os"
	"path"
	"strings"
	"github.com/russross/blackfriday"
)

//ShowPath takes a fully rooted path as an argument, and generates an HTML webpage in order in order to allow the user to navigate or clone via http. It expects the given URL to have a trailing "/".
func ShowPath(url string, p string, host string) (page string) {
	css, err := ioutil.ReadFile("style.css")
	if err != nil {
		return
	}

	//Retrieve information about the file.
	fi, err := os.Stat(p)
	if err != nil {
		//If there is an error, present
		//a 404.
		//TODO create an actual error
		return "404"
	}
	//If is not directory, or starts with ".", or is not globally readable...
	if !fi.IsDir() || strings.HasPrefix(fi.Name(), ".") || fi.Mode()&0005 == 0 {
		//Return 403 unauthorized.
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
	dirinfos, err := f.Readdir(0)
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
	for _, info := range dirinfos {
		if info.Name() == ".git" {
			isGit = true
			gitDir = info.Name()
			break
		}
	}

	if isGit {
		branch := gitBranch(p)
		sha := gitCurrentSHA(p)
		c := gitTotalCommits(p)

		html := "<html><head><title>" + userName + " [Grove]</title><style type=\"text/css\">" + string(css) + "</style></head><body><div class=\"title\"><a href=\"" + url + "..\">.. / </a>" + path.Base(p) + "<div class=\"cloneme\">git clone " + url + gitDir + "</div></div>"
		//now add the button things
		html += "<div class=\"wrapper\"><div class=\"button\"><div class=\"buttontitle\">Current Branch</div><br/><div class=\"buttontext\">" + branch + "</div></div><div class=\"button\"><div class=\"buttontitle\">Tags</div><br/><div class=\"buttontext\">3</div></div><div class=\"button\"><div class=\"buttontitle\">Commits</div><br/><div class=\"buttontext\">" + c + "</div></div><div class=\"button\"><div class=\"buttontitle\">Current Commit</div><br/><div class=\"buttontext\">" + sha + "</div></div></div>"
		//add the md
		html += "<div class=\"md\">"+md(p)+"</div>"
		//add the log
		html += "<div class=\"log\">log</div>"
		//now everything else for right now
		html += "</body></html>"

		return html
	} else {
		var dirList string = "<ul>"
		if url != ("http://" + host + "/") {
			dirList += "<a href=\"" + url + "..\"><li>..</li></a>"
		}
		for _, info := range dirinfos {
			//If is directory, and does not start with '.', and is globally readable
			if (info.IsDir()) && !strings.HasPrefix(info.Name(), ".") && (info.Mode()&0005 == 0005) {
				dirList += "<a href=\"" + url + info.Name() + "\"><li>" + info.Name() + "</li></a>"
			}
		}
		page = "<html><head><style type=\"text/css\">" + string(css) + "</style></head><body><a href=\"http://" + host + "\"><div class=\"logo\"></div></a>" + dirList + "</ul><div class=\"version\">" + Version + "</body></html>"
	}
	return
}

func md(path string) (string) {
	readme, err := ioutil.ReadFile(path+"/README.md")
	println(path+"README.md")
	if err != nil {
		return ""
	}
	return string(blackfriday.MarkdownCommon(readme))
}
