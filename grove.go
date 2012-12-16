package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"path"
	"strings"
)

const (
	Version     = "0.3.2"
	DefaultPort = "8860"
)

const (
	usage = "usage: %s [repositorydir]\n"
)

var (
	g *GitBackendHandler
)

type GitBackendHandler struct {
	Handler *cgi.Handler
	Logger  *log.Logger
}

func main() {
	logger := log.New(os.Stdout, "", log.Ltime)

	var repodir string
	if len(os.Args) > 1 {
		repodir = os.Args[1]
		if !path.IsAbs(repodir) {
			wd, err := os.Getwd()
			if err != nil {
				logger.Fatalln("Error getting working directory:", err)
			}
			path.Join(wd, repodir)
		}
	} else {
		wd, err := os.Getwd()
		if err != nil {
			logger.Fatalln("Error getting working directory:", err)
		}
		repodir = wd
	}

	err := gitVars() //Make sure that the execPath is known
	if err != nil {
		logger.Fatalln("Error getting git variables:", err)
	}

	Serve(logger, repodir, DefaultPort)
}

func Serve(logger *log.Logger, repodir string, port string) {
	g = &GitBackendHandler{
		Handler: &cgi.Handler{
			Path:   strings.TrimRight(string(execPath), "\r\n") + "/" + gitHttpBackend,
			Root:   "/",
			Dir:    repodir,
			Env:    []string{"GIT_PROJECT_ROOT=" + repodir, "GIT_HTTP_EXPORT_ALL=TRUE"},
			Logger: logger,
		},
		Logger: logger,
	}
	logger.Println("Created CGI handler:",
		"\n\tPath:\t", g.Handler.Path,
		"\n\tRoot:\t", g.Handler.Root,
		"\n\tDir:\t", g.Handler.Dir,
		"\n\tEnv:\t",
		"\n\t\t", g.Handler.Env[0],
		"\n\t\t", g.Handler.Env[1])

	logger.Println("Starting server")
	http.HandleFunc("/", HandleWeb)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		logger.Fatalln("Server crashed:", err)
	}
	return
}

func HandleWeb(w http.ResponseWriter, req *http.Request) {
	//Determine the path from the URL
	urlp := req.URL.String()
	if !strings.HasSuffix(urlp, "/") {
		urlp += "/"
	}
	path := path.Join(g.Handler.Dir, req.URL.String())
	urlp = "http://" + req.Host + urlp

	//Send the request to the git http backend
	//if it is to a .git URL.
	if strings.Contains(req.URL.String(), ".git") {
		gitPath := strings.SplitAfter(path, ".git")[0]
		g.Logger.Println("Git request to", req.URL, "from", req.RemoteAddr)

		//Check to make sure that the repository
		//is globally readable.
		fi, err := os.Stat(gitPath)
		if err != nil || !(fi.Mode()&0005 == 0005) {
			g.Logger.Println("Git request from", req.RemoteAddr, "denied")
			return
		}

		g.Handler.ServeHTTP(w, req)
		return
	} else if req.URL.String() == "/favicon.ico" {
		b, err := ioutil.ReadFile("img/favicon.png")
		if err != nil {
			return
		}
		w.Write(b)
	} else {
		g.Logger.Println("View of", req.URL, "from", req.RemoteAddr)
	}

	w.Write([]byte(ShowPath(urlp, path, req.Host)))
}
