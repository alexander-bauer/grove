package main

// Copyright ⓒ 2013 Alexander Bauer and Luke Evers (see LICENSE.md)

import (
	"encoding/base64"
	"errors"
	"github.com/russross/blackfriday"
	"html"
	"html/template"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
)

type gitPage struct {
	Prefix     string // URL prefix to be prepended
	Owner      string
	InRepoPath string
	URL        string
	GitDir     string
	Branch     string
	RootLink   string
	TagNum     string
	Path       string
	CommitNum  string
	SHA        string
	Content    template.HTML
	List       []*dirList
	Logs       []*gitLog
	Version    string
	Query      template.URL
	Status     string
}

type gitLog struct {
	Author    string
	Classtype string
	SHA       string
	Time      string
	Subject   template.HTML
	Body      template.HTML
}

type dirList struct {
	URL   template.URL
	Name  string
	Link  string
	Query template.URL
}

const (
	defaultRef     = "HEAD" // Default git reference
	defaultCommits = 10     // Default number of commits to show
)

var (
	internalServerError = errors.New(
		http.StatusText(http.StatusInternalServerError))
	forbidden = errors.New(
		http.StatusText(http.StatusForbidden))
	notFound = errors.New(
		http.StatusText(http.StatusNotFound))
)

// Check for a .git directory in the repository argument. If one does
// not exist, we will generate a directory listing, rather than a
// repository view.
func isGit(repository string) (git bool, gitDir string) {
	_, err := os.Stat(path.Join(repository, ".git"))
	if err == nil {
		// Note that err EQUALS nil
		git = true
		gitDir = ".git"
	}
	return
}

// MakePage acts as a multiplexer for the various complex http
// functions. It handles logging and web error reporting.
func MakePage(w http.ResponseWriter, req *http.Request, repository string, file string, isFile bool) {
	g := &git{
		Path: repository,
	}
	// First, establish the template and fill out some of the gitPage.
	pageinfo := &gitPage{
		Prefix:     prefix,
		Owner:      gitVarUser(),
		InRepoPath: path.Join(path.Base(repository), file),
		Path:       repository[len(handler.Dir):] + "/", // Path without in-git
		Version:    Version,
	}
	if len(*fHost) > 0 {
		pageinfo.RootLink = "http://" + *fHost
	} else {
		pageinfo.RootLink = "http://" + req.Host
	}
	pageinfo.URL = prefix + strings.TrimRight(
		req.URL.Path, "/") + "/" // Full URL with assured trailing slash

	// If there is a query, add it to the relevant field. Otherwise,
	// leave it blank.
	if len(req.URL.RawQuery) > 0 {
		pageinfo.Query = template.URL("?" + req.URL.RawQuery)
	}

	// Now, check if the given directory is a git repository, and if
	// so, parse some of the possible http forms.
	var ref string
	var maxCommits int
	git, gitDir := isGit(repository)
	if git {
		// ref is the git commit reference. If the form is not submitted,
		// (or is invalid), it is set to "HEAD".
		ref = req.FormValue("ref")
		if len(ref) == 0 || !g.RefExists(ref) {
			ref = "HEAD" // The commit or branch reference
		}

		// The form value since is just a shortcut for
		// "?ref=<ref>..<since>", so we check it here. Note that the
		// results will include <ref> and exclude <since>.
		if since := req.FormValue("since"); g.RefExists(since) {
			ref = since + ".." + ref
		}

		// maxCommits is the maximum number of commits to be loaded via
		// the log.
		var err error
		maxCommits, err = strconv.Atoi(req.FormValue("c"))
		if err != nil {
			maxCommits = 10
		}

		// Now, switch to using the API if it is requested. We access
		// req.Form directly because the form can be empty. (In this
		// case, we would fall back to checking the Accept field in
		// the header.)
		if _, useAPI := req.Form["api"]; useAPI {
			err = ServeAPI(w, req, g, ref, maxCommits)
			if err != nil {
				l.Errf("API request %q from %q failed: %s",
					req.URL, req.RemoteAddr, err)
			} else {
				l.Debugf("API request %q from %q\n",
					req.URL, req.RemoteAddr)
			}
			return
		}

		pageinfo.Branch = g.Branch("HEAD")
		pageinfo.TagNum = strconv.Itoa(len(g.Tags()))
		pageinfo.CommitNum = strconv.Itoa(g.TotalCommits())
		pageinfo.SHA = g.SHA(ref)
		pageinfo.GitDir = gitDir
	}

	// TODO: all of the below case blocks may misbehave if the URL
	// contains a keyword.
	var err error
	var status int
	switch {
	case !git:
		// This will catch all non-git cases, eliminating the need for
		// them below.
		err, status = MakeDirPage(w, pageinfo, repository)
	case strings.Contains(req.URL.Path, "/tree/"):
		// This will catch cases needing to serve directories within
		// git repositories.
		err, status = MakeTreePage(w, pageinfo, g, ref, file)
	case strings.Contains(req.URL.Path, "/blob/"):
		// This will catch cases needing to serve files.
		err, status = MakeFilePage(w, pageinfo, g, ref, file)
	case strings.Contains(req.URL.Path, "/raw/"):
		// This will catch cases needing to serve files directly.
		err, status = MakeRawPage(w, file, ref, g)
	case git:
		// This will catch cases serving the main page of a repository
		// directory. This needs to be last because the above cases
		// for "tree" and "blob" will also have `git` as true.
		err, status = MakeGitPage(w, pageinfo, g, ref, file, maxCommits)
	}

	// If an error was encountered, ensure that an error page is
	// displayed, then close the connection and return.
	if err != nil {
		l.Errf("View of %q from %q caused error: %s",
			req.URL.Path, req.RemoteAddr, err)
		Error(w, status)
	} else {
		l.Debugf("View of %q from %q\n",
			req.URL.Path, req.RemoteAddr)
	}
}

// Error reports an error of the given status to the given http
// connection using http.StatusText().
func Error(w http.ResponseWriter, status int) {
	pageinfo := &gitPage{
			Owner:      gitVarUser(),
			Status:     strconv.Itoa(status)+" - "+http.StatusText(status),
			Version:    Version,
		}
		
	t.ExecuteTemplate(w, "error.html", pageinfo)
}

func MakeAboutPage(w http.ResponseWriter) {
	pageinfo := &gitPage{
		Owner:      gitVarUser(),
		Version:    Version,
	}
	
	t.ExecuteTemplate(w, "about.html", pageinfo)
}

// MakeRawPAge makes the raw page of which the files are shown as
// completely raw files.
func MakeRawPage(w http.ResponseWriter, file, ref string, g *git) (err error, status int) {
	f := g.GetFile(ref, file)
	if len(f) == 0 {
		// If the file is not retrieved from git, return the error.
		return notFound, http.StatusNotFound
	}
	// If it is found, write the contents to the connection directly.
	w.Write(f)
	return
}

// MakeDirPage makes filesystem directory listings, which are not
// contained within git projects. It writes the webpage to the
// provided http.ResponseWriter.
func MakeDirPage(w http.ResponseWriter, pageinfo *gitPage, directory string) (err error, status int) {

	// First, check the permissions of the file to be displayed.
	fi, err := os.Stat(directory)
	if err != nil {
		return err, http.StatusNotFound
	}
	if !CheckPerms(fi) {
		return forbidden, http.StatusForbidden
	}
	// We only get beyond this point if we are allowed to serve the
	// directory.

	// We begin the template here so that we can fill it out.

	pageinfo.List = make([]*dirList, 0, 2)
	if pageinfo.Path != "/" {
		// If we're not on the root directory, we need two links for
		// navigation: "/" and ".."
		pageinfo.List = append(pageinfo.List,
			&dirList{ // append "/"
				URL:  template.URL(prefix + "/"),
				Name: "/",
			}, &dirList{ // and append ".."
				URL:  template.URL(prefix + pageinfo.Path + "../"),
				Name: "..",
			})
	}

	// Open the file so that it can be read.
	f, err := os.Open(directory)
	if err != nil {
		return err, http.StatusNotFound
	}

	// To list the directory properly, we have to do it in two
	// steps. First, retrieve the names, then perform os.Stat() on the
	// result. This is so that simlinks are followed. We will also
	// check file permissions.
	dirnames, err := f.Readdirnames(0)
	f.Close()
	if err != nil {
		return err, http.StatusInternalServerError
	}
	// We have the directory names; go on to calling os.Stat() and
	// checking their permissions. If they should be listed, add
	// them to a buffer, then append that to the dirlist at the
	// end.
	dirbuf := make([]*dirList, 0, len(dirnames))
	for _, n := range dirnames {
		info, err := os.Stat(directory + "/" + n)
		if err == nil && CheckPerms(info) {
			dirbuf = append(dirbuf, &dirList{
				URL: template.URL(prefix + pageinfo.Path +
					info.Name() + "/"),
				Name: info.Name(),
			})

		}
	}
	pageinfo.List = append(pageinfo.List, dirbuf...)

	// We return 500 here because the error will only be reported
	// if t.ExecuteTemplate() results in an error.
	return t.ExecuteTemplate(w, "dir.html", pageinfo),
		http.StatusInternalServerError
}

// MakeFilePage shows the contents of a file within a git project. It
// writes the webpage to the provided http.ResponseWriter.
func MakeFilePage(w http.ResponseWriter, pageinfo *gitPage, g *git, ref string, file string) (err error, status int) {
	// First we need to get the content,
	pageinfo.Content = template.HTML(string(g.GetFile(ref, file)))
	if len(pageinfo.Content) == 0 {
		// If there is no content, return an error.
		return notFound, http.StatusNotFound
	}
	// then we need to figure out how many lines there are.
	lines := strings.Count(string(pageinfo.Content), "\n")
	// For each of the lines, we want to prepend
	//    <div id=\"L-"+j+"\">
	// and append
	//    </div>
	// Also, we want to add line numbers.
	temp := ""
	temp_html := ""
	temp_content := strings.SplitAfter(string(pageinfo.Content), "\n")

	// Image support
	if extention := path.Ext(file); extention == ".png" ||
		extention == ".jpg" ||
		extention == ".jpeg" ||
		extention == ".gif" {

		var image []byte = []byte(pageinfo.Content)
		img := base64.StdEncoding.EncodeToString(image)
		temp_html = "<img src=\"data:image/" + strings.TrimLeft(extention, ".") + ";base64," + img + "\"/>"
	} else {
		for j := 1; j <= lines+1; j++ {
			temp_html += "<div id=\"L-" + strconv.Itoa(j) + "\">" +
				html.EscapeString(temp_content[j-1]) + "</div>"
			temp += "<a href=\"#L-" + strconv.Itoa(j) + "\" class=\"line\">" +
				strconv.Itoa(j) + "</a><br/>"
		}
	}

	pageinfo.Content = template.HTML(temp_html)

	// We return 500 here because the error will only be reported
	// if t.ExecuteTemplate() results in an error.
	return t.ExecuteTemplate(w, "file.html", pageinfo),
		http.StatusInternalServerError

}

// MakeGitPage shows the "front page" that is the main directory of a
// git reposiory, including the README and a directory listing. It
// writes the webpage to the provided http.ResponseWriter.
func MakeGitPage(w http.ResponseWriter, pageinfo *gitPage, g *git, ref, file string, maxCommits int) (err error, status int) {
	// Parse the log to retrieve the commits.
	commits := g.Commits(ref, maxCommits)

	pageinfo.Logs = make([]*gitLog, len(commits))
	for i, c := range commits {
		if len(c.SHA) == 0 {
			// If, for some reason, the commit doesn't have content,
			// skip it.
			continue
		}
		var classtype string
		if c.Author == pageinfo.Owner {
			classtype = "-owner"
		}

		pageinfo.Logs[i] = &gitLog{
			Author:    c.Author,
			Classtype: classtype,
			SHA:       c.SHA,
			Time:      c.Time,
			Subject:   template.HTML(html.EscapeString(c.Subject)),
			Body:      template.HTML(strings.Replace(html.EscapeString(c.Body), "\n", "<br/>", -1)),
		}
	}

	if len(file) == 0 {
		// Load the README if it can be located. To locate, go through
		// a list of possible names and break the loop at the first
		// one.
		for _, fn := range []string{"README", "README.txt", "README.md"} {
			readme := g.GetFile(ref, fn)
			if len(readme) != 0 {
				pageinfo.Content = template.HTML(
					blackfriday.MarkdownCommon(readme))
				break
			}
		}
	}

	// We return 500 here because the error will only be reported
	// if t.ExecuteTemplate() results in an error.
	return t.ExecuteTemplate(w, "gitpage.html", pageinfo),
		http.StatusInternalServerError
}

// MakeTreePage makes directory listings from within git repositories.
// It writes the webpage to the provided http.ResponseWriter.
func MakeTreePage(w http.ResponseWriter, pageinfo *gitPage, g *git, ref, file string) (err error, status int) {
	// Retrieve the list of files from the repository.
	files := g.GetDir(ref, file)

	// If there are no files, return an error.
	if len(files) == 0 {
		return notFound, http.StatusNotFound
	} // Otherwise, continue as normal.

	pageinfo.List = make([]*dirList, len(files))
	for n, f := range files {
		d := &dirList{
			URL:  template.URL(f)+pageinfo.Query,
			Name: f,
		}

		var t string
		if strings.HasSuffix(f, "/") {
			t = "tree"
		} else {
			t = "blob"
		}
		d.Link = prefix + pageinfo.Path + t + "/" + path.Join(file, f) + string(pageinfo.Query)
		pageinfo.List[n] = d
	}

	// We return 500 here because the error will only be reported
	// if t.ExecuteTemplate() results in an error.
	return t.ExecuteTemplate(w, "tree.html", pageinfo),
		http.StatusInternalServerError
}
