package main

import (
	"github.com/russross/blackfriday"
	"html"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
)

//ShowPath takes a fully rooted path as an argument, and generates an HTML webpage in order in order to allow the user to navigate or clone via http. It expects the given URL not to have a trailing "/".
func ShowPath(url, p, host string) (page string, status int) {
	//Create (or retrieve, if caching is possible) a
	//git object.
	p = strings.SplitN(p, "?", 2)[0]
	g := &git{
		Path: p,
	}

	ref := "HEAD"       //The commit or branch reference
	maxCommits := 10    //The maximum number of commits to be shown by the log
	file := "README.md" //The file to display in the WebUI
	jsoni := false      //Whether or not to use the JSON interface
	//Parse out variables, such as in:
	//    http://host/path/to/repo?o=deadbeef
	//Keys are:
	//    r: ref, such as SHA or branch name
	components := strings.Split(url, "?")
	for i, c := range components {
		if i == 0 {
			//The first component is always the url,
			//and trim out any trailing slashes.
			url = strings.TrimRight(c, "/")
			continue
		}

		parts := strings.SplitN(c, "=", 2)
		var name string
		var val string
		if len(parts) > 0 {
			name = strings.ToLower(parts[0])
		}
		if len(parts) > 1 {
			val = parts[1]
		}
		switch name {
		case "r":
			if g.RefExists(val) {
				ref = val
			}
		case "c":
			tmax, err := strconv.Atoi(val)
			if err != nil {
				continue
			}
			maxCommits = tmax
		case "f":
			file = val
		case "j":
			jsoni = true
		}
	}

	//Retrieve information about the file.
	fi, err := os.Stat(p)
	if err != nil {
		//If there is an error, present
		//a StatusNotFound.
		return page, http.StatusNotFound
	}
	//If is not directory, or starts with ".", or is not readable...
	if !fi.IsDir() || !CheckPerms(fi) {
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
	var isGit bool
	var gitDir string
	for _, info := range dirinfos {
		if info.Name() == ".git" {
			isGit = true
			gitDir = info.Name()
			break
		}
	}

	//If the request is specified as using the JSON interface,
	//then we switch to that.
	if jsoni && isGit {
		return g.ShowJSON(ref, maxCommits)
	}

	//Otherwise, load the CSS.
	css, err := ioutil.ReadFile(*fRes + "/style.css")
	if err != nil {
		return page, http.StatusInternalServerError
	}

	if isGit {
		owner := gitVarUser()
		commits := g.Commits(ref, 0)
		commitNum := len(commits)
		tagNum := len(g.Tags())
		branch := g.Branch("HEAD")
		sha := g.SHA(ref)

		HTML := "<html><head><title>" + owner + " [Grove]</title><style type=\"text/css\">" + string(css) + "</style></head><body><div class=\"title\"><a href=\"" + url + "/..\">.. / </a>" + path.Base(p) + "<div class=\"cloneme\">" + url + gitDir + "</div></div>"
		//now add the button things
		HTML += "<div class=\"wrapper\"><div class=\"button\"><div class=\"buttontitle\">Developer's Branch</div><br/><div class=\"buttontext\">" + branch + "</div></div><div class=\"button\"><div class=\"buttontitle\">Tags</div><br/><div class=\"buttontext\">" + strconv.Itoa(tagNum) + "</div></div><div class=\"button\"><div class=\"buttontitle\">Commits</div><br/><div class=\"buttontext\">" + strconv.Itoa(commitNum) + "</div></div><div class=\"button\"><div class=\"buttontitle\">Grove View</div><br/><div class=\"buttontext\">" + sha + "</div></div></div>"
		//add the file, usually README
		if strings.Contains(file, "README") {
			HTML += "<div class=\"md\">" + getREADME(g, ref, file) + "</div>"
		} else {
			HTML += "<div class=\"md\"><pre>" + strings.Replace(string(g.GetFile(ref, file)), "\n", "<br/>", -1) + "</pre></div>"
		}
		//add the log
		HTML += "<div class=\"log\">"

		for i, c := range commits {
			if len(c.SHA) == 0 {
				//If, for some reason, the commit doesn't
				//have content, skip it.
				continue
			}

			var classtype string
			if c.Author == owner {
				classtype = "-owner"
			}

			HTML += "<div class=\"loggy" + classtype + "\">"
			HTML += c.Author + " &mdash; <div class=\"SHA" + classtype + "\">" + c.SHA + "</div> &mdash; " + c.Time + "<br/>"
			HTML += "<br/><strong><div class=\"holdem\">" + html.EscapeString(c.Subject) + "</strong><br/><br/>"
			HTML += strings.Replace(html.EscapeString(c.Body), "\n", "<br/>", -1) + "</div></div>"
			if i == maxCommits-1 {
				//but only display certain log messages
				break
			}
		}
		//now everything else for right now
		HTML += "</div></body></html>"

		return HTML, http.StatusOK
	} else {
		var dirList string = "<ul>"
		if url != ("http://" + host + "/") {
			dirList += "<a href=\"" + url + "/..\"><li>..</li></a>"
		}
		for _, info := range dirinfos {
			//If is directory, and does not start with '.', and is globally readable
			if info.IsDir() && CheckPerms(info) {
				dirList += "<a href=\"" + url + "/" + info.Name() + "\"><li>" + info.Name() + "</li></a>"
			}
		}
		page = "<html><head><title>" + gitVarUser() + " [Grove]</title></head><style type=\"text/css\">" + string(css) + "</style></head><body><a href=\"http://" + host + "\"><div class=\"logo\"></div></a>" + dirList + "</ul><div class=\"version\">" + Version + minversion + "</body></html>"
	}
	return page, http.StatusOK
}

func getREADME(g *git, ref, file string) string {
	readme := g.GetFile(ref, file)
	return string(blackfriday.MarkdownCommon(readme))
}
