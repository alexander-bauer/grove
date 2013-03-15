// Grove - Git self-hosting for developers
//
// Copyright ⓒ 2013 Alexander Bauer and Luke Evers (GPLv3)
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http/cgi"
	"os"
	"path"
)

var (
	Version    = "0.5.5"
	minversion string

	Bind      = "0.0.0.0"          // Interface to bind to
	Port      = "8860"             // Port to bind to
	Resources = "/usr/share/grove" // Directory to store resources in
)

var (
	l       *log.Logger
	handler *cgi.Handler
)

const (
	usage = "usage: %s [repositorydir]\n"
)

var (
	fBind = flag.String("bind", Bind, "interface to bind to")
	fPort = flag.String("port", Port, "port to listen on")
	fRes  = flag.String("res", Resources, "resources directory")

	fShowVersion  = flag.Bool("version", false, "print major version and exit")
	fShowFVersion = flag.Bool("version-full", false, "print full version and exit")
	fShowBind     = flag.Bool("show-bind", false, "print default bind interface and exit")
	fShowPort     = flag.Bool("show-port", false, "print default port and exit")
	fShowRes      = flag.Bool("show-res", false, "print default resources directory and exit")
)

func main() {
	l = log.New(os.Stdout, "", log.Ltime)

	flag.Parse()

	switch {
	case *fShowVersion:
		fmt.Fprintln(os.Stdout, Version)
		return
	case *fShowFVersion:
		fmt.Fprintln(os.Stdout, Version+minversion)
		return
	case *fShowBind:
		fmt.Fprintln(os.Stdout, Bind)
		return
	case *fShowPort:
		fmt.Fprintln(os.Stdout, Port)
		return
	case *fShowRes:
		fmt.Fprintln(os.Stdout, Resources)
		return
	}

	l.Println("Verision:", Version+minversion)

	var repodir string
	if flag.NArg() > 0 {
		repodir = path.Clean(flag.Arg(0))

		if !path.IsAbs(repodir) {
			wd, err := os.Getwd()
			if err != nil {
				l.Fatalln("Error getting working directory:", err)
			}
			repodir = path.Join(wd, repodir)
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
