package main

import (
	"bytes"
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
	TagNum    string
	CommitNum string
	SHA       string
	Content   template.HTML
	List      []*dirList
	Logs      []*gitLog
	Location  template.URL
	Numbers   string
}

type gitLog struct {
	Author    string
	Classtype string
	SHA       string
	Time      string
	Subject   string
	Body      template.HTML
}

type dirList struct {
	URL   template.URL
	Name  string
	Class string
}

//ShowPath takes a fully rooted path as an argument, and generates an HTML webpage in order in order to allow the user to navigate or clone via http. It makes no assumptions regarding the presence of a trailing slash.
func ShowPath(url, p, host string) (page string, status int) {
	//Create (or retrieve, if caching is possible) a
	//git object.

	p = strings.SplitN(p, "?", 2)[0]
	g := &git{
		Path: p,
	}

	ref := "HEAD"    //The commit or branch reference
	maxCommits := 10 //The maximum number of commits to be shown by the log
	var file string  //The file to display in the WebUI
	jsoni := false   //Whether or not to use the JSON interface
	//Parse out variables, such as in:
	//    http://host/path/to/repo?r=deadbeef
	//Keys are:
	//    r: ref, such as SHA or branch name
	//    c: number of commits to display
	//    f: file or directory to browse (directories have a trailing slash)
	//    j: use the JSON interface if present
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

	pageinfo := &gitPage{
		Owner:     owner,
		CSS:       template.CSS(css),
		BasePath:  path.Base(p),
		URL:       url,
		GitDir:    gitDir,
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
				//If, for some reason, the commit doesn't
				//have content, skip it.
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
				Subject:   html.EscapeString(c.Subject),
				Body:      template.HTML(strings.Replace(html.EscapeString(c.Body), "\n", "<br/>", -1)),
			})
			if i == maxCommits-1 {
				//but only display certain log messages
				break
			}
		}
		pageinfo.Logs = Logs
		//view readme
		if len(file) == 0 {
			pageinfo.Content = template.HTML(getREADME(g, ref, "README.md"))
			t, _ = template.ParseFiles(*fRes + "/templates" + "/gitpage.html")
		} else {
			//view directory
			pageinfo.Location = template.URL("/" + file)
			if strings.HasSuffix(file, "/") {
				List := make([]*dirList, 0)
				files := g.GetDir(ref, file)
				for _, f := range files {
					/*
						// This code is to specify file or directory but does not work with
						// the information I have available in this spot, I think...
								List = append(List, dirList{
									URL:   template.URL("?f=" + file + f),
									Name:  f,
									Class: "dir",
								}) 

							if f.IsDir() {
								List = append(List, dirList{
									URL:   template.URL("?f=" + file + f),
									Name:  info.Name(),
									Class: "dir",
								})
							} else {
					*/
					List = append(List, &dirList{
						URL:   template.URL("?f=" + file + f),
						Name:  f,
						Class: "file",
					})
					//}
				}

				pageinfo.List = List
				t, _ = template.ParseFiles(*fRes + "/templates" + "/dir.html")
			} else {
				//view file
				pageinfo.Content = template.HTML(html.EscapeString(string(g.GetFile(ref, file))))
				
				i := strings.Count(string(pageinfo.Content), "\n")
				temp := ""
				for j := 1; j <= i; j++ {
					temp += strconv.Itoa(j)+" \n "
				}
				
				pageinfo.Numbers = temp
				
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
				URL:   template.URL(url),
				Name:  "..",
				Class: "dir",
			})
		}
		for _, info := range dirinfos {
			//If is directory, and does not start with '.', and is globally readable
			if CheckPerms(info) {
				if info.IsDir() {
					List = append(List, &dirList{
						URL:   template.URL(info.Name() + "/"),
						Name:  info.Name(),
						Class: "dir",
					})
				} else {
					List = append(List, &dirList{
						URL:   template.URL(info.Name()),
						Name:  info.Name(),
						Class: "file",
					})
				}
			}

		}
		//println(*fRes)
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

func getREADME(g *git, ref, file string) string {
	readme := g.GetFile(ref, file)
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