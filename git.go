package main

import (
	"os/exec"
	"strconv"
	"strings"
)

type Commit struct {
	SHA     string //Full SHA of the commit
	Author  string //Author of the commit
	Time    string //Relative time of the commit
	Subject string //Subject of the commit
	Body    string //Body of the commit
}

const (
	gitHttpBackend = "git-http-backend"
	gitLogFmt      = "%H%n%cr%n%an%n%s%n%b"
)

var (
	execPath string //The path containing git binaries
)

func setExecPath() (err error) {
	//Use 'git --exec-path' to get the path
	//of the git executables.
	path, err := execute("", "git", "--exec-path")
	execPath = strings.TrimRight(path, "\n")
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

func gitCommit(ref string, path string) (commit *Commit) {
	log, _ := execute(path, "git", "--no-pager", "log", "--format=format:'"+gitLogFmt+"'", ref, "-n 1")
	return gitParseCommit(strings.Split(log, "\n"))
}

//Log formats, as given by gitLogFmt, should be as follows.
//    <full hash>
//    <commit time relative>
//    <author name>
//    <nonwrapped commit message>
func gitParseCommit(log []string) (commit *Commit) {
	if len(log) < 4 {
		return
	}
	commit = &Commit{
		SHA:     log[0],
		Time:    log[1],
		Author:  log[2],
		Subject: log[3],
	}
	for i := 0; i < len(log)-4; i++ {
		commit.Body += log[i]
	}
	return
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
