package main

// Copyright ⓒ 2013 Alexander Bauer and Luke Evers (see LICENSE.md)

import (
	"bytes"
	"encoding/base64"
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
	Owner     string
	BasePath  string
	URL       string
	GitDir    string
	Branch    string
	Host      string
	TagNum    string
	Path      string
	CommitNum string
	SHA       string
	Content   template.HTML
	List      []*dirList
	Logs      []*gitLog
	Location  template.URL
	Numbers   template.HTML
	Version   string
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
	URL      template.URL
	Name     string
	Class    string
	Type     string
	Host     string
	Path     string
	Location string
	Version  string
}

// getREADME is a utility function which retrieves the given file from
// the repository at a particular ref, HTML escapes it, converts any
// markdown to HTML, and returns it as a string. It is intended for
// use with READMEs, but could potentially be used for other files.
func getREADME(g *git, ref, file string) string {
	readme := g.GetFile(ref, file)
	readme = []byte(html.EscapeString(string(readme)))
	return string(blackfriday.MarkdownCommon(readme))
}

// Check for a .git directory in the repository argument. If one does
// not exist, we will generate a directory listing, rather than a
// repository view.
func isGit(repository string) (git bool, gitDir string) {
	_, err := os.Stat(path.Join(repository, ".git"))
	if err == nil {
		// Note that err EQUALS nil
		git = true
		gitDir = "/.git"
	}
	return
}

// Retrieval of file info is done in two steps so that we can use
// os.Stat(), rather than os.Lstat(), the former of which follows
// symlinks.
func MakeDirInfos(repository string, dirnames []string) (dirinfos []os.FileInfo) {
	dirinfos = make([]os.FileInfo, 0, len(dirnames))
	for _, n := range dirnames {
		info, err := os.Stat(repository + "/" + n)
		if err == nil && CheckPerms(info) {
			dirinfos = append(dirinfos, info)
		}
	}
	return
}

func MakePage(req *http.Request, repository string, file string, isFile bool) (page string, status int) {
	g := &git{
		Path: repository,
	}

	url := "http://" + req.Host + strings.TrimRight(req.URL.Path, "/")

	// ref is the git commit reference. If the form is not submitted,
	// (or is invalid), it is set to "HEAD".
	ref := req.FormValue("r")
	if len(ref) == 0 || !g.RefExists(ref) {
		ref = "HEAD" // The commit or branch reference
	}

	// maxCommits is the maximum number of commits to be loaded via
	// the log.
	maxCommits, err := strconv.Atoi(req.FormValue("c"))
	if err != nil {
		maxCommits = 10
	}

	// jsoni is a boolean indicator of whether or not to use the json
	// interface.
	jsoni := strings.ToLower(req.FormValue("j")) == "true"

	// If the request is specified as using the JSON interface, then
	// we switch to that. This usually isn't done, but it is better to
	// do it here than to wait until the dirinfos are retrieved.
	git, gitDir := isGit(repository)
	if jsoni && git {
		return g.ShowJSON(ref, maxCommits)
	}

	// If we're doing a directory listing, then we need to retrieve
	// the directory list.
	var dirinfos []os.FileInfo
	if !git {
		// Open the file so that it can be read.
		f, err := os.Open(repository)
		if err != nil || f == nil {
			// If there is an error opening the file, return 500.
			return page, http.StatusInternalServerError
		}
		dirnames, err := f.Readdirnames(0)
		f.Close()
		if err != nil {
			// If the directory could not be opened, return 500.
			return page, http.StatusInternalServerError
		}
		dirinfos = MakeDirInfos(repository, dirnames)
	}

	// Get the user.name from the git config
	owner := gitVarUser()

	var commits []*Commit
	if len(file) != 0 {
		commits = g.CommitsByFile(ref, file, maxCommits)
	} else {
		commits = g.Commits(ref, maxCommits)
	}

	commitNum := len(commits)
	tagNum := len(g.Tags())
	branch := g.Branch("HEAD")
	sha := g.SHA(ref)

	var doc bytes.Buffer
	t := template.New("Grove!")

	// Set up the gitPage template.
	pathto := strings.SplitAfter(string(repository), handler.Dir)
	pageinfo := &gitPage{
		Owner:     owner,
		BasePath:  path.Base(repository),
		URL:       url,
		GitDir:    gitDir,
		Host:      req.Host,
		Version:   Version,
		Path:      pathto[1],
		Branch:    branch,
		TagNum:    strconv.Itoa(tagNum),
		CommitNum: strconv.Itoa(commitNum),
		SHA:       sha,
		Location:  template.URL(""),
	}

	// TODO: all of the below case blocks may misbehave if the URL
	// contains a keyword.
	switch {
	case !git:
		// This will catch all non-git cases, eliminating the need for
		// them below.
		return MakeDirPage(t, doc, pageinfo, req, file, url, dirinfos),
			http.StatusOK
	case strings.Contains(req.URL.Path, "tree"):
		// This will catch cases needing to serve directories within
		// git repositories.
		return MakeTreePage(t, doc, pageinfo, req, file, url,
			g, ref, pathto), http.StatusOK
	case strings.Contains(req.URL.Path, "blob"):
		// This will catch cases needing to serve files.
		return MakeFilePage(t, doc, pageinfo, g, ref, file),
			http.StatusOK
	case git:
		// This will catch cases serving the main page of a repository
		// directory. This needs to be last because the above cases
		// for "tree" and "blob" will also have `git` as true.
		return MakeGitPage(t, doc, pageinfo, ref, g, commits,
				owner, maxCommits, file),
			http.StatusOK
	default:
		// Finally, if this case is reached, something is very wrong.
		return "", http.StatusInternalServerError
	}
	return
}

// MakeDirPage makes filesystem directory listings, which are not
// contained within git projects. It returns an entire webpage as a
// string.
func MakeDirPage(t *template.Template, doc bytes.Buffer, pageinfo *gitPage, req *http.Request, file string, url string, dirinfos []os.FileInfo) string {
	pageinfo.Location = template.URL("/" + file)
	List := make([]*dirList, 0)
	if url != ("http://" + req.Host + "/") {
		List = append(List, &dirList{
			URL:   template.URL(url + ".."),
			Name:  "..",
			Class: "dir",
		})
	}

	// If is directory, and does not start with '.', and is globally
	// readable
	for _, info := range dirinfos {
		if info.IsDir() && CheckPerms(info) {
			List = append(List, &dirList{
				URL:   template.URL(info.Name() + "/"),
				Name:  info.Name(),
				Class: "dir",
			})
		}
	}

	pageinfo.List = List
	t, _ = template.ParseFiles(*fRes + "/templates" + "/dir.html")

	return Execute(t, doc, pageinfo)
}

// MakeFilePage shows the contents of a file within a git project. It
// returns an entire webpage as a string.
func MakeFilePage(t *template.Template, doc bytes.Buffer, pageinfo *gitPage, g *git, ref string, file string) (page string) {
	// First we need to get the content,
	pageinfo.Content = template.HTML(string(g.GetFile(ref, file)))
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
			temp_html += "<div id=\"L-" + strconv.Itoa(j) + "\">" + html.EscapeString(temp_content[j-1]) + "</div>"
			temp += "<a href=\"#L-" + strconv.Itoa(j) + "\" class=\"line\">" + strconv.Itoa(j) + "</a><br/>"
		}
	}

	pageinfo.Numbers = template.HTML(temp)
	pageinfo.Content = template.HTML(temp_html)

	// Finally, parse it.
	t, _ = template.ParseFiles(*fRes + "/templates" + "/file.html")
	return Execute(t, doc, pageinfo)
}

// MakeGitPage shows the "front page" that is the main directory of a
// git reposiory, including the README and a directory listing. It
// returns an entire webpage as a string.
func MakeGitPage(t *template.Template, doc bytes.Buffer, pageinfo *gitPage, ref string, g *git, commits []*Commit, owner string, maxCommits int, file string) (page string) {
	Logs := make([]*gitLog, 0)
	for i, c := range commits {
		if len(c.SHA) == 0 {
			// If, for some reason, the commit doesn't have content,
			// skip it.
			continue
		}
		var classtype string
		if c.Author == owner {
			classtype = "-owner"
		}

		Logs = append(Logs, &gitLog{
			Author:    c.Author,
			Classtype: classtype,
			SHA:       c.SHA,
			Time:      c.Time,
			Subject:   template.HTML(html.EscapeString(c.Subject)),
			Body:      template.HTML(strings.Replace(html.EscapeString(c.Body), "\n", "<br/>", -1)),
		})
		if i == maxCommits-1 {
			// but only display certain log messages
			break
		}
	}
	pageinfo.Logs = Logs
	if len(file) == 0 {
		// Load the README
		pageinfo.Content = template.HTML(getREADME(g, ref, "README"))
		pageinfo.Content = template.HTML(getREADME(g, ref, "README.md"))
		t, _ = template.ParseFiles(*fRes + "/templates" + "/gitpage.html")
	}
	return Execute(t, doc, pageinfo)
}

// MakeTreePage makes directory listings from within git repositories.
// It returns an entire webpage as a string.
func MakeTreePage(t *template.Template, doc bytes.Buffer, pageinfo *gitPage, req *http.Request, file string, url string, g *git, ref string, pathto []string) (page string) {
	pageinfo.Location = template.URL("/" + file)
	if strings.HasSuffix(file, "/") {
		List := make([]*dirList, 0)
		files := g.GetDir(ref, file)
		for _, f := range files {
			if strings.HasSuffix(f, "/") {
				List = append(List, &dirList{
					URL:      template.URL(f),
					Type:     "tree",
					Host:     req.Host,
					Path:     pathto[1],
					Name:     f,
					Location: file,
					Version:  Version,
					Class:    "file",
				})
			} else {
				List = append(List, &dirList{
					URL:      template.URL(f),
					Type:     "blob",
					Name:     f,
					Host:     req.Host,
					Path:     pathto[1],
					Location: file,
					Class:    "file",
					Version:  Version,
				})
			}
		}
		pageinfo.List = List
		t, _ = template.ParseFiles(*fRes + "/templates" + "/tree.html")
	}
	return Execute(t, doc, pageinfo)
}

// Execute executes a template (using html/template) and returns the
// result as a string.
func Execute(t *template.Template, doc bytes.Buffer, pageinfo *gitPage) string {
	err := t.Execute(&doc, pageinfo)
	if err != nil {
		l.Println(err)
		return http.StatusText(http.StatusInternalServerError)
	}
	return doc.String()
}
