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

type git struct {
	Path string //Directory path
}

//Set a number of git variables.
func gitVarExecPath() (execPath string) {
	//Use 'git --exec-path' to get the path
	//of the git executables.
	g := &git{}
	execPath, _ = g.execute("--exec-path")
	execPath = strings.TrimRight(execPath, "\n")
	return
}

func gitVarUser() (user string) {
	//Use 'git config --global user.name
	//to retrieve the variable.
	g := &git{}
	user, _ = g.execute("config", "--global", "user.name")
	user = strings.TrimRight(user, "\n")
	return
}

func (g *git) Branch(ref string) (branch string) {
	branch, _ = g.execute("rev-parse", "--abbrev-ref", ref)
	return strings.TrimRight(branch, "\n")
}

//Retrieve the contents of a file from the repository. The commit is either a SHA or pointer (such as HEAD, or HEAD^).
func (g *git) GetFile(commit, file string) (contents []byte) {
	contents, _ = g.executeB("--no-pager", "show", commit+":"+file)
	return contents
}

//Retrieve a list of items in a directory from the repository. The commit is either a SHA or a pointer (such as HEAD, or HEAD^).
func (g *git) GetDir(commit, dir string) (files []string) {
	output, _ := g.execute("--no-pager", "show", "--name-only", commit+":"+dir)
	parts := strings.SplitN(output, "\n\n", 2) //Split on the blank line
	if len(parts) == 2 && strings.HasPrefix(parts[0], "tree") {
		return strings.Split(strings.TrimRight(parts[1], "\n"), "\n")
	}
	println(output)
	return
}

func (g *git) SHA(ref string) (sha string) {
	commit, _ := g.execute("rev-parse", "--short=8", ref)
	return strings.TrimRight(commit, "\n")
}

func (g *git) Tags() (tags []string) {
	t, _ := g.execute("tag", "--list")
	return strings.Split(strings.TrimRight(t, "\n"), "\n")
}

func (g *git) TotalCommits() (commits string) {
	c, _ := g.execute("rev-list", "--all")
	commit := strings.Split(strings.TrimRight(c, "\n"), "\n")
	return strconv.Itoa(len(commit))
}

func (g *git) RefExists(ref string) (exists bool) {
	//If the exit status of 'git rev-list HEAD..<ref>'
	//is nonzero, the ref does not exist in the
	//repository. Cmd.Output(), which is used by
	//execute(), uses Cmd.Run(), which returns an
	//error if an exit status other than 0 is returned.
	_, err := g.execute("rev-list", "HEAD.."+ref)
	return err == nil
}

//Get Commits from the log, up to the given max.
func (g *git) Commits(ref string, max int) (commits []*Commit) {
	var log string
	if max > 0 {
		log, _ = g.execute("--no-pager", "log", "--format=format:"+gitLogFmt+gitLogSep, ref, "-n "+strconv.Itoa(max))
	} else {
		log, _ = g.execute("--no-pager", "log", "--format=format:"+gitLogFmt+gitLogSep, ref)
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
func (g *git) execute(args ...string) (output string, err error) {
	out, err := g.executeB(args...)
	return string(out), err
}

func (g *git) executeB(args ...string) (output []byte, err error) {
	cmd := exec.Command("git", args...)
	if len(g.Path) != 0 {
		cmd.Dir = g.Path
	}
	out, err := cmd.Output()
	return out, err
}
