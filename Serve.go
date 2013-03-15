package main

// Copyright ⓒ 2013 Alexander Bauer (see LICENSE.md)

import (
	"net/http"
	"net/http/cgi"
	"os"
	"path"
	"strconv"
	"strings"
)

var (
	Perms = uint(0)
	// Used to specify which files can be served:
	// 0: readable globally
	// 1: readable by group
	// 2: readable
)

// Serve creates an HTTP server using net/http and initializes it
// appropriately.
func Serve(repodir string) {
	handler = &cgi.Handler{
		Path: gitVarExecPath() + "/" + gitHttpBackend,
		Root: "/",
		Dir:  repodir,
		Env: []string{"GIT_PROJECT_ROOT=" + repodir,
			"GIT_HTTP_EXPORT_ALL=TRUE"},
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
	http.HandleFunc("/res/style.css", HandleCSS)
	http.HandleFunc("/favicon.ico", HandleIcon)
	err := http.ListenAndServe(*fBind+":"+*fPort, nil)
	if err != nil {
		l.Fatalln("Server crashed:", err)
	}
	return
}

// HandleCSS uses http.ServeFile() to serve `style.css` directly from
// the file system.
func HandleCSS(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, path.Join(*fRes, "style.css"))
}

// HandleIcon uses http.ServeFile() to serve the favicon directly from
// the filesystem.
func HandleIcon(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, path.Join(*fRes, "favicon.png"))
}

// HandleWeb handles general requests, such as for the web interface
// or git-over-http requests.
func HandleWeb(w http.ResponseWriter, req *http.Request) {
	// Determine the filesystem path from the URL.
	p := path.Join(handler.Dir, req.URL.Path)

	// Send the request to the git http backend if it is to a .git
	// URL.
	if strings.Contains(req.URL.String(), ".git/") {
		gitPath := strings.SplitAfter(p, ".git/")[0]
		l.Println("Git request to", req.URL, "from", req.RemoteAddr)

		// Check to make sure that the repository is globally
		// readable.
		fi, err := os.Stat(gitPath)
		if err != nil || !CheckPermBits(fi) {
			l.Println("Git request from", req.RemoteAddr, "denied")
			return
		}

		handler.ServeHTTP(w, req)
		return
	}
	l.Println("View of", req.URL, "from", req.RemoteAddr)

	// Figure out which directory is being requested, and check
	// whether we're allowed to serve it.
	repository, file, isFile, status := SplitRepository(handler.Dir, p)
	if status == http.StatusOK {
		var body string
		body, status = MakePage(req, repository, file, isFile)
		if status == http.StatusOK {
			w.Write([]byte(body))
			return
		}
	}

	// If ShowPath gives the status as anything other than 200 OK,
	// write the error in the header.
	l.Println("Sending", req.RemoteAddr, "status:", status)
	http.Error(w, "Could not serve "+req.URL.String()+"\n"+strconv.Itoa(status), status)
}

// SplitRepository checks each directory in the path (p), traversing
// upward, until it finds a .git folder. If the parent directory of
// this .git directory is not permissable to serve (globally readable
// and listable, by default), or a .git directory could not be found,
// or the path is invalid, this function will return an appropriate
// exit code.  This function will only recurse upward until it reaches
// the path indicated by toplevel.
func SplitRepository(toplevel, p string) (repository, file string, isFile bool, status int) {
	path.Clean(toplevel)
	// Set the repository to the path for the moment, to simplify the
	// loop
	repository = p
	i := 0
	for {
		// We behave differently on the first run through, so only do
		// this step if i is not 0.
		if i != 0 {
			// Traverse upward.
			file = path.Join(path.Base(repository), file)
			repository = path.Dir(repository)
		}

		// Check if we shouldn't continue.
		if repository == toplevel {
			repository = path.Join(repository, file)
			file = ""
			status = http.StatusOK
			return
		}

		// Check if the path has a .git folder.
		_, err := os.Stat(repository + "/.git")
		if err != nil {
			// If not, traverse up and start again.
			i++
			continue
		}

		// If the .git directory was discovered, then we now have to
		// check if we are allowed to serve the parent directory.
		fi, err := os.Stat(repository)
		if err != nil {
			// An error at this point would imply that the server is
			// in error.
			status = http.StatusInternalServerError
			return
		}

		// If all is well, check if it's servable.
		if !CheckPerms(fi) {
			// If not, 403 Forbidden.
			status = http.StatusForbidden
			return
		}

		// If the file is prefixed with /blob/, then treat it as a
		// file. If it has /tree/, then treat it as a directory. In
		// either case, chop off the prefix.  If it has neither, 404.
		if len(file) != 0 {
			// The trailing slash trickery involves avoiding runtime
			// errors and splitting the strings sanely.
			file += "/"
			if strings.HasPrefix(file, "blob/") {
				file = strings.SplitAfterN(file, "/", 2)[1]
				isFile = true
				file = strings.TrimRight(file, "/")
			} else if strings.HasPrefix(file, "tree/") {
				// Remove the /tree/, but be sure that, if the file is
				// blank, to make it "/" instead.
				file = strings.SplitAfterN(file, "/", 2)[1]
				if len(file) == 0 {
					file = "./"
				}
			} else {
				status = http.StatusNotFound
				return
			}
		}
		status = http.StatusOK
		return
	}
	// Something is very wrong if we get here.
	status = http.StatusInternalServerError
	return
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

	// For example, consider the following:
	// 
	//       rwl rwl rwl       r-l
	//    0b 111 101 101 & (0b 101 << 3)  > 0
	//    0b 111 101 101 & 0b 000 101 000 > 0
	//    0b 000 101 000                  > 0
	//    TRUE
	// 
	// Thus, the file is readable and listable by the group, and
	// therefore okay to serve.
	return (info.Mode().Perm()&os.FileMode((permBits<<(Perms*3))) > 0)
}
