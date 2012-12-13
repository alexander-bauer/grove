package main

import (
	//"net/http"
	"log"
	"os"
)

const (
	DefaultPort = 8860
)

const (
	usage = "usage: %s [repositorydir] [logfile]\n"
)

func main() {
	logger := log.New(os.Stdout, "", log.Ltime)

	wd, err := os.Getwd()
	if err != nil {
		logger.Fatalln("Error getting working directory:", err)
	}

	Serve(logger, wd, DefaultPort)
}

func Serve(logger *log.Logger, repodir string, port int) {
	logger.Println("Repository directory is:", repodir)
	logger.Println("Binding to port:", port)
	logger.Println("Starting server")
}
