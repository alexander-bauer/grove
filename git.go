package main

import (
	"os/exec"
	"strconv"
	"strings"
)

const (
	gitHttpBackend = "git-http-backend"
)

var (
	execPath string //The path containing git binaries
	userName string //The user.name global variable
)

//Set a number of git variables.
func gitVars() (err error) {
	err = setExecPath()
	if err != nil {
		return
	}
	err = setUser()
	if err != nil {
		return
	}
	return
}

func setExecPath() (err error) {
	//Use 'git --exec-path' to get the path
	//of the git executables.
	path, err := execute("", "git", "--exec-path")
	execPath = strings.TrimRight(path, "\n")
	return
}

func setUser() (err error) {
	//Use 'git config --global user.name
	//to retrieve the variable.
	name, err := execute("", "git", "config", "--global", "user.name")
	userName = strings.TrimRight(name, "\n")
	return
}

func gitBranch(path string) (branch string) {
	branch, _ = execute(path, "git", "rev-parse", "--abbrev-ref", "HEAD")
	return strings.TrimRight(branch, "\n")
}

func gitCurrentSHA(path string) (sha string) {
	commit, _ := execute(path, "git", "rev-parse", "--short=8", "HEAD")
	return strings.TrimRight(commit, "\n")
}

func gitTotalCommits(path string) (commits string) {
	c, _ := execute(path, "git", "rev-list", "--all")
	commit := strings.Split(strings.TrimRight(c, "\n"), "\n")
	return strconv.Itoa(len(commit))
}

//Execute invokes exec.Command() with the given command, arguments, and working directory. All CR ('\r') characters are removed in output.
func execute(dir, command string, args ...string) (output string, err error) {
	cmd := exec.Command(command, args...)
	if len(dir) != 0 {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	return string(out), err
}
