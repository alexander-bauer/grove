// Grove - Git self-hosting for developers
//
// Copyright ⓒ 2013 Alexander Bauer and Luke Evers (GPLv3)
package main

import (
	"flag"
	"github.com/inhies/go-utils/log"
	_ "log"
	"os"
	"path"
)

var (
	Version = "0.5.11"

	Bind      = "0.0.0.0"          // Interface to bind to
	Port      = "8860"             // Port to bind to
	Resources = "/usr/share/grove" // Directory to store resources in
	BaseURL   = ""                 // Hostname and prefix to use in links

	LogLevel log.LogLevel = log.INFO // Default log level
)

var (
	l *log.Logger
)

const (
	usage = "usage: %s [repositorydir]\n"
)

var (
	fQuiet = flag.Bool("q", false, "disable logging output")
	//	fVerbose = flag.Bool("v", false, "enable verbose output")
	fDebug = flag.Bool("debug", false, "enable debugging output")

	fBind = flag.String("bind", Bind, "interface to bind to")
	fPort = flag.String("port", Port, "port to listen on")
	fRes  = flag.String("res", Resources, "resources directory")
	fHost = flag.String("host", BaseURL,
		"hostname and prefix to use in links")
	fWeb = flag.Bool("web", true, "enable web browsing")

	fShowVersion  = flag.Bool("version", false, "print major version and exit")
	fShowFVersion = flag.Bool("version-full", false, "print full version and exit")
	fShowBind     = flag.Bool("show-bind", false, "print default bind interface and exit")
	fShowPort     = flag.Bool("show-port", false, "print default port and exit")
	fShowRes      = flag.Bool("show-res", false, "print default resources directory and exit")
)

func main() {
	flag.Parse()

	// Open a new logger with an appropriate log level.
	if *fQuiet {
		LogLevel = -1 // Disable ALL output
		//	} else if *fVerbose {
		//		LogLevel = log.INFO
	} else if *fDebug {
		LogLevel = log.DEBUG
	}
	l, _ = log.NewLevel(LogLevel, true, os.Stdout, "", log.Ltime)

	// If any of the 'show' flags are set, print the relevant variable
	// and exit.
	switch {
	case *fShowVersion:
		l.Println(Version)
		return
	case *fShowFVersion:
		l.Println(Version)
		return
	case *fShowBind:
		l.Println(Bind)
		return
	case *fShowPort:
		l.Println(Port)
		return
	case *fShowRes:
		l.Println(Resources)
		return
	}

	l.Infof("Starting Grove version %s\n", Version)

	var repodir string
	if flag.NArg() > 0 {
		repodir = path.Clean(flag.Arg(0))

		if !path.IsAbs(repodir) {
			wd, err := os.Getwd()
			if err != nil {
				l.Fatalf("Error getting working directory: %s\n", err)
			}
			repodir = path.Join(wd, repodir)
		}
	} else {
		wd, err := os.Getwd()
		if err != nil {
			l.Fatalf("Error getting working directory: %s\n", err)
		}
		repodir = wd
	}

	Serve(repodir)
}
