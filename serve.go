package main

// Copyright ⓒ 2013 Alexander Bauer and Luke Evers (see LICENSE.md)

import (
	"compress/gzip"
	"html/template"
	"io"
	"net/http"
	"net/http/cgi"
	"os"
	"path"
	"strings"
)

var (
	Perms = uint(0)
	// Used to specify which files can be served:
	// 0: readable globally
	// 1: readable by group
	// 2: readable

	prefix       string // Path to prepend to links
	prefixLength int    // Number of characters to strip from requests

	handler *cgi.Handler       // git-http-backend CGI handler
	t       *template.Template // Template containing all webui templates

	templateFiles = []string{ // Basenames of the HTML templates
		"dir.html", "file.html",
		"gitpage.html", "tree.html",
		"error.html", "about.html",
	}
)

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
	detectDone bool
}

// Serve creates an HTTP server using net/http and initializes it
// appropriately. If the fWeb flagg is true, it will serve directory
// trees and git repositories to incoming requests.
func Serve(repodir string) {
	handler = &cgi.Handler{
		Path: gitVarExecPath() + "/" + gitHttpBackend,
		Root: "/",
		Dir:  repodir,
		Env: []string{"GIT_PROJECT_ROOT=" + repodir,
			"GIT_HTTP_EXPORT_ALL=TRUE"},
		Logger: &l.Logger,
	}

	// Set up the stripProxy variable, but only if *fHost contains a
	// path to strip, such as "example.com/grove"
	if hostLength := strings.Index(*fHost, "/"); hostLength > 0 {
		prefixLength = len(*fHost) - hostLength
		prefix = (*fHost)[hostLength:]
	}

	var err error
	t, err = getTemplate()
	if err != nil {
		l.Emerg("HTML templates failed to load; exiting\n")
		return
	} else {
		l.Debug("Templates loaded successfully\n")
	}

	l.Infof("Starting server on %s:%s\n", *fBind, *fPort)
	l.Infof("Serving %q\n", repodir)
	l.Infof("Web access: %t\n", *fWeb)
	
	// Regardless if fWeb is true or not, host the CSS
	http.HandleFunc(prefix+"/res/style.css", gzipHandler(HandleCSS))
	
	if *fWeb {
		http.HandleFunc(prefix+"/res/highlight.js", gzipHandler(HandleJS))
		http.HandleFunc(prefix+"/favicon.ico", gzipHandler(HandleIcon))
		http.HandleFunc("/", gzipHandler(HandleWeb))
	} else {
		http.HandleFunc("/", gzipHandler(HandleAbout))
	}

	err = http.ListenAndServe(*fBind+":"+*fPort, nil)
	if err != nil {
		l.Fatalf("Server crashed: %s", err)
	}
	return
}

// HandleJS uses http.ServeFile() to serve `highlight.js` directly
// from the file system.
func HandleJS(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, path.Join(*fRes, "highlight.js"))
}

// HandleCSS uses http.ServeFile() to serve `style.css` directly from
// the file system.
func HandleCSS(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, path.Join(*fRes, "light.css"))
}

// HandleIcon uses http.ServeFile() to serve the favicon directly from
// the filesystem.
func HandleIcon(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, path.Join(*fRes, "favicon.png"))
}

// HandleAbout makes an about page to be served regardless of the path
// that the user is trying to look at. This func is only to be used as
// a handler when *fWeb is true.
func HandleAbout(w http.ResponseWriter, req *http.Request) {
	l.Noticef("Web access denied to %q\n", req.RemoteAddr)
	MakeAboutPage(w)
}

// HandleWeb handles general requests, such as for the web interface
// or git-over-http requests.
func HandleWeb(w http.ResponseWriter, req *http.Request) {
	// Determine the filesystem path from the URL. We must first make
	// sure that we strip the prefix, if appropriate. We do this by
	// modifying the http.Request directly.
	if len(req.URL.Path) < prefixLength {
		// If the request URL is shorter than the prefix, (which will
		// never occur when the prefix is not specified), then throw
		// an error.
		Error(w, http.StatusBadRequest)
		return
	} else {
		req.URL.Path = req.URL.Path[prefixLength:]
	}
	p := path.Join(handler.Dir, req.URL.Path)

	// Send the request to the git http backend if it is to a .git
	// URL.
	if strings.Contains(req.URL.String(), ".git/") {
		gitPath := strings.SplitAfter(p, ".git/")[0]
		l.Debugf("Git request to %q from %q\n",
			req.URL, req.RemoteAddr)

		// Check to make sure that the repository is globally
		// readable.
		fi, err := os.Stat(gitPath)
		if err != nil {
			l.Errf("Git request of %q from %q produced error: %s\n",
				req.URL.Path, req.RemoteAddr, err)
			http.NotFound(w, req)
			return
		}
		if !CheckPermBits(fi) {
			l.Noticef("Git request to %q from %q denied\n",
				req.URL.Path, req.RemoteAddr)
			http.Error(w, http.StatusText(http.StatusForbidden),
				http.StatusForbidden)
			return
		}

		handler.ServeHTTP(w, req)
		return
	}

	// Figure out which directory is being requested, and check
	// whether we're allowed to serve it.
	repository, file, isFile, status := SplitRepository(handler.Dir, p)
	if status == http.StatusOK {
		MakePage(w, req, repository, file, isFile)
	}
}

// If the client accepts gzipped responses, that's what we'll send,
// otherwise use the default http handler to send data.
func gzipHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fn(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		fn(gzipResponseWriter{Writer: gz, ResponseWriter: w}, r)
	}
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.detectDone {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", http.DetectContentType(b))
		}
		w.detectDone = true
	}
	return w.Writer.Write(b)
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
			} else if strings.HasPrefix(file, "raw/") {
				file = strings.SplitAfterN(file, "/", 2)[1]
				isFile = true
				file = strings.TrimRight(file, "/")
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

// getTemplate uses the global variables templateFiles and *fRes to
// load the templates and return the given object.
func getTemplate() (t *template.Template, err error) {
	// First, ensure that the paths are correct.
	files := make([]string, len(templateFiles))
	for i, f := range templateFiles {
		files[i] = path.Join(*fRes, "templates", f)
	}
	// Now, return the results.
	return template.New("master").ParseFiles(files...)
}
