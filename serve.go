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

	handler *cgi.Handler       // git-http-backend CGI handler
	t       *template.Template // Template containing all webui templates

	templateFiles = []string{ // Basenames of the HTML templates
		"dir.html", "file.html",
		"gitpage.html", "tree.html",
		"error.html", "about.html",
	}

	prefixLength int // Length of *fPrefix
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
	user = (&git{repodir}).User()

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
	l.Infof("Username: %s\n", user)
	l.Infof("Prefix: %s", *fPrefix)
	l.Infof("Web access: %t\n", *fWeb)
	l.Infof("Theme: %s", *fTheme)

	// Set the prefixLength variable, for easy use in the future.
	prefixLength = len(*fPrefix)

	// Set up the appropriate handlers depending on whether web
	// browsing is enabled or not.
	http.HandleFunc(*fPrefix+"/res/", HandleRes)
	
	if *fWeb {
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

// HandleRes handles everything inside of the resources directory
// (which is res/ by default). In order to take into consideration the
// fact that it might not be always just in res/ we have to do some go
// magic.
func HandleRes (w http.ResponseWriter, req *http.Request) {
	s := strings.Split(req.URL.Path, "/")
	s[0] = ""
	s[1] = ""
	p := strings.Join(s, "/")
	http.ServeFile(w, req, path.Join(*fPrefix, *fRes, p))
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

	repository, file, g, isDir, status := AnalyzePath(handler.Dir,
		p, req.Form.Get("ref"))
	if status == http.StatusOK {
		MakePage(w, req, g, repository, file, isDir)
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

// AnalyzePath uses the provided information and appropriate git calls
// to split apart the given path "p" into the containing repository,
// file within that repository, whether that file is a directory, and
// the appropriate http status. It will return g if "p" points to a
// path within a git repository, such that g.Path is the top level of
// that repository, and nil if it does not.
func AnalyzePath(toplevel, p, ref string) (repository, file string, g *git, isDir bool, status int) {
	toplevel, p = path.Clean(toplevel), path.Clean(p)

	// l will be the length of the path which represents the
	// repository level which is being checked.
	l := len(p)

	g = &git{}
	// Loop through until the repository is set.
	for {
		g.Path = p[:l]
		repository = g.TopLevel()

		// If we encounter an error, such as the file not existing,
		// then we modify l to move the path up one directory, and
		// then, if the next path is appropriate, continue. If it is
		// not, go on to the next check and allow it to return.
		if len(repository) == 0 && l > len(toplevel) {
			l = strings.LastIndex(g.Path, "/")
			if l > len(toplevel) { // implies l != -1
				continue
			}
		}

		// If we do not encounter an error, but find the lowest level
		// repository which we are in to be above the top level, then
		// we must behave as if we did not find one.
		if l < len(toplevel) || len(repository) < len(toplevel) {
			repository = p
			g = nil
			status = http.StatusOK
			return
		}

		if len(repository) > 0 && len(repository) > len(toplevel) {
			// If the repository was discovered, then we now have to
			// check if we are allowed to serve the parent directory.
			fi, err := os.Stat(repository)
			if err != nil {
				// An error at this point would imply that the server
				// is in error.
				status = http.StatusInternalServerError
				return
			}

			// If all is well, check if it's servable.
			if !CheckPerms(fi) {
				// If not, 403 Forbidden.
				status = http.StatusForbidden
				return
			}

			// If it can be served, split off the rest of the path and
			// set the file to be returned.
			file = strings.TrimLeft(p[len(repository):], "/")
			println(file)

			// Next, check the status of the file. We must sanitize
			// the ref, if possible.
			if !g.RefExists(ref) {
				ref = "HEAD"
			}

			isDir, err = g.IsDir(ref, file)
			if err != nil {
				// If there is an error at this point, the file
				// probably does not exist at the given ref.
				status = http.StatusNotFound
			}

			// Set up g so that it can used properly. Note that it is
			// *not* reallocated from earlier.
			g.Path = repository

			// If everything up to this point has been executed
			// properly, we can set the status as OK and return.
			status = http.StatusOK
			return
		}
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
