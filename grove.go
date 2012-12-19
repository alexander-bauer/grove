package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"path"
	"strconv"
	"strings"
)

const (
	Version     = "0.4.3"
	DefaultPort = "8860"
)

const (
	usage = "usage: %s [repositorydir]\n"
)

var (
	Bind   = ""          //Bind interface (such as 127.0.0.1)
	Port   = DefaultPort //Port to listen on
	ResDir = "res/"      //Resources directory
)

var (
	l       *log.Logger
	handler *cgi.Handler
)

func main() {
	l = log.New(os.Stdout, "", log.Ltime)

	var repodir string
	if len(os.Args) > 1 {
		repodir = os.Args[1]
		if !path.IsAbs(repodir) {
			wd, err := os.Getwd()
			if err != nil {
				l.Fatalln("Error getting working directory:", err)
			}
			path.Join(wd, repodir)
		}
	} else {
		wd, err := os.Getwd()
		if err != nil {
			l.Fatalln("Error getting working directory:", err)
		}
		repodir = wd
	}

	err := gitVars() //Make sure that the execPath is known
	if err != nil {
		l.Fatalln("Error getting git variables:", err)
	}

	Serve(repodir)
}

func Serve(repodir string) {
	handler = &cgi.Handler{
		Path:   strings.TrimRight(string(execPath), "\r\n") + "/" + gitHttpBackend,
		Root:   "/",
		Dir:    repodir,
		Env:    []string{"GIT_PROJECT_ROOT=" + repodir, "GIT_HTTP_EXPORT_ALL=TRUE"},
		Logger: l,
	}

	l.Println("Created CGI handler:",
		"\n\tPath:\t", handler.Path,
		"\n\tRoot:\t", handler.Root,
		"\n\tDir:\t", handler.Dir,
		"\n\tEnv:\t",
		"\n\t\t", handler.Env[0],
		"\n\t\t", handler.Env[1])

	l.Println("Starting server on", Bind+":"+Port)
	http.HandleFunc("/", HandleWeb)
	err := http.ListenAndServe(Bind+":"+Port, nil)
	if err != nil {
		l.Fatalln("Server crashed:", err)
	}
	return
}

func HandleWeb(w http.ResponseWriter, req *http.Request) {
	//Determine the path from the URL
	urlp := req.URL.String()
	if !strings.HasSuffix(urlp, "/") {
		urlp += "/"
	}
	path := path.Join(handler.Dir, req.URL.String())
	urlp = "http://" + req.Host + urlp

	//Send the request to the git http backend
	//if it is to a .git URL.
	if strings.Contains(req.URL.String(), ".git") {
		gitPath := strings.SplitAfter(path, ".git")[0]
		l.Println("Git request to", req.URL, "from", req.RemoteAddr)

		//Check to make sure that the repository
		//is globally readable.
		fi, err := os.Stat(gitPath)
		if err != nil || !(fi.Mode()&0005 == 0005) {
			l.Println("Git request from", req.RemoteAddr, "denied")
			return
		}

		handler.ServeHTTP(w, req)
		return
	} else if req.URL.String() == "/favicon.ico" {
		b, err := ioutil.ReadFile(ResDir + "favicon.png")
		if err != nil {
			return
		}
		w.Write(b)
		return
	} else {
		l.Println("View of", req.URL, "from", req.RemoteAddr)
	}
	body, status := ShowPath(urlp, path, req.Host)

	//If ShowPath gives the status as anything
	//other than 200 OK, write the error in the
	//header.
	if status != http.StatusOK {
		l.Println("Sending", req.RemoteAddr, "status:", status)
		http.Error(w, "Could not serve "+req.URL.String()+"\n"+strconv.Itoa(status), status)
	} else {
		w.Write([]byte(body))
	}
}
