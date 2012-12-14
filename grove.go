package main

import (
	"io"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"strings"
)

const (
	Version     = "0.0"
	DefaultPort = "8860"
)

const (
	usage = "usage: %s [repositorydir] [logfile]\n"
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

	wd, err := os.Getwd()
	if err != nil {
		logger.Fatalln("Error getting working directory:", err)
	}

	Serve(logger, wd, DefaultPort)
}

func Serve(logger *log.Logger, repodir string, port string) (err error) {
	g = &GitBackendHandler{
		Handler: &cgi.Handler{
			Path:   "git http-backend",
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
	err = http.ListenAndServe(":"+port, nil)
	return
}

func HandleWeb(w http.ResponseWriter, req *http.Request) {
	//Send the request to the git http backend
	//if it is to a .git URL.
	if strings.HasSuffix(req.URL.String(), ".git") {
		g.Logger.Println("Git request to", req.URL, "from", req.RemoteAddr)
		g.Handler.ServeHTTP(w, req)
		return
	} else {
		g.Logger.Println("View of", req.URL, "from", req.RemoteAddr)
	}

	io.WriteString(w, "<html>Welcome to <a href=\"https://github.com/SashaCrofter/grove\">grove</a>.")
}
