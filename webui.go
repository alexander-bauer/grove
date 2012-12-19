package main

import (
	"github.com/russross/blackfriday"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"regexp"
)

//ShowPath takes a fully rooted path as an argument, and generates an HTML webpage in order in order to allow the user to navigate or clone via http. It expects the given URL to have a trailing "/".
func ShowPath(url string, p string, host string) (page string, status int) {
	css, err := ioutil.ReadFile(ResDir + "style.css")
	if err != nil {
		return page, http.StatusInternalServerError
	}

	//Retrieve information about the file.
	fi, err := os.Stat(p)
	if err != nil {
		//If there is an error, present
		//a StatusNotFound.
		return page, http.StatusNotFound
	}
	//If is not directory, or starts with ".", or is not globally readable...
	if !fi.IsDir() || strings.HasPrefix(fi.Name(), ".") || fi.Mode()&0005 == 0 {
		//Return 403 forbidden.
		return page, http.StatusForbidden
	}

	f, err := os.Open(p)
	if err != nil || f == nil {
		//If there is an error opening
		//the file, return 500.
		return page, http.StatusInternalServerError
	}

	//Retrieval of file info is done in two steps
	//so that we can use os.Stat(), rather than
	//os.Lstat(), the former of which follows
	//symlinks.
	dirnames, err := f.Readdirnames(0)
	f.Close()
	if err != nil {
		//If the directory could not be
		//opened, return 500.
		return page, http.StatusInternalServerError
	}
	dirinfos := make([]os.FileInfo, 0, len(dirnames))
	for _, n := range dirnames {
		info, err := os.Stat(p + "/" + n)
		if err == nil {
			dirinfos = append(dirinfos, info)
		}
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
		phtml := regexp.MustCompile("<|>")
		commits := gitCommits("HEAD", 0, p)
		commitNum := len(commits)
		tagNum := gitTotalTags(p)
		branch := gitBranch(p)
		sha := gitCurrentSHA(p)

		html := "<html><head><title>" + userName + " [Grove]</title><style type=\"text/css\">" + string(css) + "</style></head><body><div class=\"title\"><a href=\"" + url + "..\">.. / </a>" + path.Base(p) + "<div class=\"cloneme\">" + url + gitDir + "</div></div>"
		//now add the button things
		html += "<div class=\"wrapper\"><div class=\"button\"><div class=\"buttontitle\">Current Branch</div><br/><div class=\"buttontext\">" + branch + "</div></div><div class=\"button\"><div class=\"buttontitle\">Tags</div><br/><div class=\"buttontext\">" + strconv.Itoa(tagNum) + "</div></div><div class=\"button\"><div class=\"buttontitle\">Commits</div><br/><div class=\"buttontext\">" + strconv.Itoa(commitNum) + "</div></div><div class=\"button\"><div class=\"buttontitle\">Current Commit</div><br/><div class=\"buttontext\">" + sha + "</div></div></div>"
		//add the md
		html += "<div class=\"md\">" + md(p) + "</div>"
		//add the log
		html += "<div class=\"log\">"
		for i := 0; i < 10; i++ {
			html += "<div class=\"loggy\">"
			html += commits[i].Author + " &mdash; <div class=\"SHA\">" + commits[i].SHA + "</div> &mdash; " + commits[i].Time + "<br/>"
			html += "<br/><strong><div class=\"holdem\">" + string(phtml.ReplaceAll([]byte(commits[i].Subject), []byte(""))) + "</strong><br/><br/>"
			html += strings.Replace(string(phtml.ReplaceAll([]byte(commits[i].Body), []byte(""))), "\n", "<br/>", -1) + "</div></div>"
		}
		//now everything else for right now
		html += "</div></body></html>"

		return html, http.StatusOK
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
		page = "<html><head><title>" + userName + " [Grove]</title></head><style type=\"text/css\">" + string(css) + "</style></head><body><a href=\"http://" + host + "\"><div class=\"logo\"></div></a>" + dirList + "</ul><div class=\"version\">" + Version + "</body></html>"
	}
	return page, http.StatusOK
}

func md(path string) string {
	readme, err := ioutil.ReadFile(path + "/README.md")
	if err != nil {
		return ""
	}
	return string(blackfriday.MarkdownCommon(readme))
}
