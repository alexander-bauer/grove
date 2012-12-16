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
	gitLogSep      = "----GROVE-LOG-SEPARATOR----"
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

func gitTotalTags(path string) (numOfTags int) {
	t, _ := execute(path, "git", "tag", "--list")
	return len(strings.Split(t, "\n"))
}

func gitTotalCommits(path string) (commits string) {
	c, _ := execute(path, "git", "rev-list", "--all")
	commit := strings.Split(strings.TrimRight(c, "\n"), "\n")
	return strconv.Itoa(len(commit))
}

//Get Commits from the log, up to the given max.
func gitCommits(ref string, max int, path string) (commits []*Commit) {
	var log string
	if max > 0 {
		log, _ = execute(path, "git", "--no-pager", "log", "--format=format:"+gitLogFmt+gitLogSep, ref, "-n "+strconv.Itoa(max))
	} else {
		//TODO THIS DOES NOT ACTUALLY GET ALL OF THE MESSAGES
		log, _ = execute(path, "git", "--no-pager", "log", "--format=format:"+gitLogFmt+gitLogSep, ref)
	}
	commitLogs := strings.Split(log, gitLogSep)
	commits = make([]*Commit, 0, len(commitLogs))
	for _, l := range commitLogs {
		commit := gitParseCommit(strings.Split(l, "\n"))
		if commit != nil {
			commits = append(commits, commit)
		}
	}
	return
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
		if log[i] != gitLogSep {
			commit.Body += log[i]
		}
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
