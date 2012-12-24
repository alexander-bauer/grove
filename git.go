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

//Set a number of git variables.
func gitVarExecPath() (execPath string) {
	//Use 'git --exec-path' to get the path
	//of the git executables.
	execPath, _ = execute("", "git", "--exec-path")
	execPath = strings.TrimRight(execPath, "\n")
	return
}

func gitVarUser() (user string) {
	//Use 'git config --global user.name
	//to retrieve the variable.
	user, _ = execute("", "git", "config", "--global", "user.name")
	user = strings.TrimRight(user, "\n")
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
	cmd := exec.Command(command, args...)
	if len(dir) != 0 {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	return string(out), err
}
