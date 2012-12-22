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

//Retrieve the contents of a file from the repository. The commit is either a SHA or pointer (such as HEAD, or HEAD^).
func gitGetFile(path, commit, file string) (contents []byte) {
	contents, _ = executeB(path, "git", "--no-pager", "show", commit+":"+file)
	return contents
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
	var sha string
	var time string
	var author string
	var subject string
	var body string

	for _, l := range log {
		if len(sha) == 0 {
			//If l is empty, then this will
			//be run again.
			sha = l
			continue
		}
		if len(time) == 0 {
			time = l
			continue
		}
		if len(author) == 0 {
			author = l
			continue
		}
		if len(subject) == 0 {
			subject = l
			continue
		}

		body += l + "\n"
	}

	commit = &Commit{
		SHA:     sha,
		Time:    time,
		Author:  author,
		Subject: subject,
		Body:    body,
	}

	return
}

//Execute invokes exec.Command() with the given command, arguments, and working directory. All CR ('\r') characters are removed in output.
func execute(dir, command string, args ...string) (output string, err error) {
	out, err := executeB(dir, command, args...)
	return string(out), err
}

func executeB(dir, command string, args ...string) (output []byte, err error) {
	cmd := exec.Command(command, args...)
	if len(dir) != 0 {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	return out, err
}
