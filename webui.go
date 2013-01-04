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

//ShowPath takes a fully rooted path as an argument, and generates an HTML webpage in order in order to allow the user to navigate or clone via http. It makes no assumptions regarding the presence of a trailing slash.
//To view a git repository, pass both a repository and a file. To view just a directory tree, leave file empty, and be sure that the repository argument is a valid directory that does not contain a .git directory.
func ShowPath(url, repository, file, queries, host string) (page string, status int) {
	g := &git{
		Path: repository,
	}

	ref := "HEAD"    //The commit or branch reference
	maxCommits := 10 //The maximum number of commits to be shown by the log
	jsoni := false   //Whether or not to use the JSON interface
	//Parse out variables, such as in:
	//    http://host/path/to/repo?r=deadbeef
	//Keys are:
	//    r: ref, such as SHA or branch name
	//    c: number of commits to display
	//    j: use the JSON interface if present
	components := strings.Split(strings.TrimLeft(queries, "?"), "?")
	for _, c := range components {
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
		case "j":
			jsoni = true
		}
	}

	//We do not need to check if we can serve the
	//repository that we've been passed. That's
	//already been done.

	//Check for a .git directory in the repository
	//argument. If one does not exist, we will
	//generate a directory listing, rather than a
	//repository view.
	var isGit bool
	var gitDir string
	_, err := os.Stat(repository + "/.git")
	if err == nil {
		//Note that if err EQUALS nil
		isGit = true
		gitDir = ".git"
	}

	//If the request is specified as using the JSON interface,
	//then we switch to that. This usually isn't done, but
	//it is better to do it here than to wait until the
	//dirinfos are retrieved.
	if jsoni && isGit {
		return g.ShowJSON(ref, maxCommits)
	}

	//Is we're doing a directory listing, then
	//we need to retrieve the directory list.
	var dirinfos []os.FileInfo
	if !isGit {
		//Open the file so that it can be read.
		f, err := os.Open(repository)
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
		dirinfos = make([]os.FileInfo, 0, len(dirnames))
		for _, n := range dirnames {
			info, err := os.Stat(repository + "/" + n)
			if err == nil && CheckPerms(info) {
				dirinfos = append(dirinfos, info)
			}
		}
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

		HTML := "<html><head><title>" + owner + " [Grove]</title><style type=\"text/css\">" + string(css) + "</style></head><body><div class=\"title\"><a href=\"" + url + "/..\">.. / </a>" + path.Base(repository) + "<div class=\"cloneme\">" + url[:len(url)-len(file)] + gitDir + "</div></div>"
		//now add the button things
		HTML += "<div class=\"wrapper\"><div class=\"button\"><div class=\"buttontitle\">Developer's Branch</div><br/><div class=\"buttontext\">" + branch + "</div></div><div class=\"button\"><div class=\"buttontitle\">Tags</div><br/><div class=\"buttontext\">" + strconv.Itoa(tagNum) + "</div></div><div class=\"button\"><div class=\"buttontitle\">Commits</div><br/><div class=\"buttontext\">" + strconv.Itoa(commitNum) + "</div></div><div class=\"button\"><div class=\"buttontitle\">Grove View</div><br/><div class=\"buttontext\">" + sha + "</div></div></div>"
		//add the file, usually README
		if len(file) == 0 {
			HTML += "<div class=\"md\">" + getREADME(g, ref, "README.md") + "</div>"
		} else {
			if strings.HasSuffix(file, "/") {
				HTML += "<div class=\"view-dir\"><ul>"
				files := g.GetDir(ref, file)
				for _, f := range files {
					HTML += "<a href=\"?f=" + f + "\"><li>" + f + "</li></a>"
				}
				HTML += "</ul></div>"
			} else {
				HTML += "<div class=\"view-file\">" + strings.Replace(string(g.GetFile(ref, file)), "\n", "<br/>", -1) + "</div>"
			}
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
