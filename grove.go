package main

import (
	"flag"
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
	Version     = "0.4.6"
	DefaultPort = "8860"
)

var (
	ResDir = "res/"  //Resources directory
	Perms  = uint(0) //Used to specify which files can be served:
	//0: readable globally
	//1: readable by group
	//2: readable
)

const (
	usage = "usage: %s [repositorydir]\n"
)

var (
	fBind = flag.String("bind", "", "interface to bind to")
	fPort = flag.String("port", DefaultPort, "port to listen on")
)

var (
	l       *log.Logger
	handler *cgi.Handler
)

func main() {
	l = log.New(os.Stdout, "", log.Ltime)

	flag.Parse()

	var repodir string
	if flag.NArg() > 0 {
		repodir = flag.Arg(0)
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

	Serve(repodir)
}

func Serve(repodir string) {
	handler = &cgi.Handler{
		Path:   gitVarExecPath() + "/" + gitHttpBackend,
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

	l.Println("Starting server on", *fBind+":"+*fPort)
	http.HandleFunc("/", HandleWeb)
	err := http.ListenAndServe(*fBind+":"+*fPort, nil)
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
		if err != nil || !CheckPermBits(fi) {
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

func CheckPerms(info os.FileInfo) (canServe bool) {
	if strings.HasPrefix(info.Name(), ".") {
		return false
	}
	return CheckPermBits(info)
}

func CheckPermBits(info os.FileInfo) (canServe bool) {
	permBits := 0004
	if info.IsDir() {
		permBits = 0005
	}

	//For example, consider the following:
	//       rwl rwl rwl       r-l
	//    0b 111 101 101 & (0b 101 << 3)  > 0
	//    0b 111 101 101 & 0b 000 101 000 > 0
	//    0b 000 101 000                  > 0
	//    TRUE
	//Thus, the file is readable and listable by
	//the group, and therefore okay to serve.
	return (info.Mode().Perm()&os.FileMode((permBits<<(Perms*3))) > 0)
}
