package main

import (
	"bytes"
	"encoding/base64"
	"github.com/russross/blackfriday"
	"html"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
)

type gitPage struct {
	Owner     string
	CSS       template.CSS
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

// ShowPath takes a fully rooted path as an argument, and generates an
// HTML webpage in order in order to allow the user to navigate or
// clone via http. It makes no assumptions regarding the presence of a
// trailing slash.  To view a git repository, pass both a repository
// and a file. To view just a directory tree, leave file empty, and be
// sure that the repository argument is a valid directory that does
// not contain a .git directory.
func ShowPath(url, repository, file string, isFile bool, queries, host string) (page string, status int) {
	g := &git{
		Path: repository,
	}

	ref := "HEAD"    // The commit or branch reference
	maxCommits := 10 // The maximum number of commits to be shown by the log
	jsoni := false   // Whether or not to use the JSON interface
	// Parse out variables, such as in:
	//    http://host/path/to/repo?r=deadbeef
	// Keys are:
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

	// We do not need to check if we can serve the repository that
	// we've been passed. That's already been done.

	// Check for a .git directory in the repository argument. If one
	// does not exist, we will generate a directory listing, rather
	// than a repository view.
	var isGit bool
	var gitDir string
	_, err := os.Stat(repository + "/.git")
	if err == nil {
		// Note that if err EQUALS nil
		isGit = true
		gitDir = ".git"
	}

	// If the request is specified as using the JSON interface, then
	// we switch to that. This usually isn't done, but it is better to
	// do it here than to wait until the dirinfos are retrieved.
	if jsoni && isGit {
		return g.ShowJSON(ref, maxCommits)
	}

	// Is we're doing a directory listing, then we need to retrieve
	// the directory list.
	var dirinfos []os.FileInfo
	if !isGit {
		// Open the file so that it can be read.
		f, err := os.Open(repository)
		if err != nil || f == nil {
			// If there is an error opening the file, return 500.
			return page, http.StatusInternalServerError
		}

		// Retrieval of file info is done in two steps so that we can
		// use os.Stat(), rather than os.Lstat(), the former of which
		// follows symlinks.
		dirnames, err := f.Readdirnames(0)
		f.Close()
		if err != nil {
			// If the directory could not be opened, return 500.
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

	// Otherwise, load the CSS.
	css, err := ioutil.ReadFile(*fRes + "/style.css")
	if err != nil {
		return page, http.StatusInternalServerError
	}
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

	pathto := strings.SplitAfter(string(repository), handler.Dir)

	pageinfo := &gitPage{
		Owner:     owner,
		CSS:       template.CSS(css),
		BasePath:  path.Base(repository),
		URL:       url,
		GitDir:    gitDir,
		Host:      host,
		Version:   Version,
		Path:      pathto[1],
		Branch:    branch,
		TagNum:    strconv.Itoa(tagNum),
		CommitNum: strconv.Itoa(commitNum),
		SHA:       sha,
		Location:  template.URL(""),
	}
	if isGit {
		Logs := make([]*gitLog, 0)
		for i, c := range commits {
			if len(c.SHA) == 0 {
				// If, for some reason, the commit doesn't have
				// content, skip it.
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
		} else {
			// or display the directory.
			pageinfo.Location = template.URL("/" + file)
			if strings.HasSuffix(file, "/") {
				List := make([]*dirList, 0)
				files := g.GetDir(ref, file)
				for _, f := range files {
					if strings.HasSuffix(f, "/") {
						List = append(List, &dirList{
							URL:      template.URL(f),
							Type:     "tree",
							Host:     host,
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
							Host:     host,
							Path:     pathto[1],
							Location: file,
							Class:    "file",
							Version:  Version,
						})
					}
				}
				pageinfo.List = List
				t, _ = template.ParseFiles(*fRes + "/templates" + "/tree.html")
			} else {
				// DON'T FUCKING TOUCH ANYTHING IN THIS ELSE BLOCK
				// YES, THAT MEANS YOU.

				// First we need to get the content
				pageinfo.Content = template.HTML(string(g.GetFile(ref, file)))
				// Then we need to figure out how many lines there are.
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
				if strings.HasSuffix(file, "png") || strings.HasSuffix(file, "jpg") || strings.HasSuffix(file, "jpeg") || strings.HasSuffix(file, "gif") {
					imagetype := "png"
					if strings.HasSuffix(file, "jpg") {
						imagetype = "jpg"
					} else if strings.HasSuffix(file, "jpeg") {
						imagetype = "jpeg"
					} else if strings.HasSuffix(file, "gif") {
						imagetype = "gif"
					}

					var image []byte = []byte(pageinfo.Content)
					img := base64.StdEncoding.EncodeToString(image)
					temp_html = "<img src=\"data:image/" + imagetype + ";base64," + img + "\"/>"
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
			}
		}

		err := t.Execute(&doc, pageinfo)
		if err != nil {
			l.Println(err)
			return page, http.StatusInternalServerError
		}

		return doc.String(), http.StatusOK
	} else {
		var doc bytes.Buffer

		pageinfo.Location = template.URL("/" + file)
		List := make([]*dirList, 0)
		if url != ("http://" + host + "/") {
			List = append(List, &dirList{
				URL:   template.URL(url + ".."),
				Name:  "..",
				Class: "dir",
			})
		}
		for _, info := range dirinfos {
			// If is directory, and does not start with '.', and is
			// globally readable
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
		err = t.Execute(&doc, pageinfo)
		if err != nil {
			l.Println(err)
			return page, http.StatusInternalServerError
		}

		return doc.String(), http.StatusOK
	}
	return page, http.StatusInternalServerError
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

func MakePage(template string, args string) (page string, status int) {
	status = http.StatusOK
	if template == "dir" {

	} else if template == "file" {

	} else if template == "gitpage" {

	} else {
		status = http.StatusInternalServerError
	}

	return
}
