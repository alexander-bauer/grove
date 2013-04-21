package main

// Copyright ⓒ 2013 Alexander Bauer and Luke Evers (see LICENSE.md)

import (
	"os/exec"
	"strconv"
	"strings"
)

type Commit struct {
	SHA     string // Full SHA of the commit
	Author  string // Author of the commit
	Email   string // Email attached to the commit
	Time    string // Relative time of the commit
	Subject string // Subject of the commit
	Body    string // Body of the commit
}

const (
	gitHttpBackend = "git-http-backend"
	gitLogFmt      = "%H%n%cr%n%an%n%ae%n%s%n%b"
	gitLogSep      = "----GROVE-LOG-SEPARATOR----"
)

type git struct {
	Path string // Directory path
}

// Set a number of git variables.
func gitVarExecPath() (execPath string) {
	// Use 'git --exec-path' to get the path of the git executables.
	g := &git{}
	execPath, _ = g.execute("--exec-path")
	execPath = strings.TrimRight(execPath, "\n")
	return
}

func gitVarUser() (user string) {
	// Use 'git config --global user.name to retrieve the variable.
	g := &git{}
	user, _ = g.execute("config", "--global", "user.name")
	user = strings.TrimRight(user, "\n")
	return
}

func (g *git) Email() (email string) {
	// Use 'git config user.email to retrieve the variable. Note that
	// it does not use '--global' so it can vary by repository.
	email, _ = g.execute("config", "user.email")
	return strings.TrimRight(email, "\n")
}

func (g *git) Branch(ref string) (branch string) {
	branch, _ = g.execute("rev-parse", "--abbrev-ref", ref)
	return strings.TrimRight(branch, "\n")
}

func (g *git) Branches() (branches []string) {
	// Retrieve a list of branches separated by "\n" and indented by
	// either two spaces or "* ".
	branchList, _ := g.execute("branch", "--no-color")
	// Prepare the slice by counting the number of newlines, including
	// the final one.
	branches = make([]string, strings.Count(branchList, "\n"))
	for n, b := range strings.Split(
		strings.TrimRight(branchList, "\n"), "\n") {
		// The call to strings.TrimLeft() will remove any number of
		// leading spaces and asterisks.
		branches[n] = strings.TrimLeft(b, "* ")
	}
	return
}

// GetBranchDescription uses git config to retrieve the branch
// description from the repository configuration file, if it's set. It
// will attempt to parse branch names from refs like
// `<oldRef>..<newRef>`.
func (g *git) GetBranchDescription(branch string) (description string) {
	// Attempt to parse the branch name if it looks like it's in the
	// form of a comparison.
	if idx := strings.LastIndex(branch, ".."); idx > -1 {
		branch = branch[idx+2:] // Add 2 to ignore the ".."
	} // Otherwise, just continue.
	output, _ := g.execute("config", "branch."+branch+".description")
	return strings.TrimRight(output, "\n")
}

// GetFile retrives the contents of a file from the repository. The
// commit is either a SHA or pointer (such as HEAD, or HEAD^).
func (g *git) GetFile(commit, file string) (contents []byte) {
	contents, _ = g.executeB("--no-pager", "show", commit+":"+file)
	return contents
}

// Retrieve a list of items in a directory from the repository. The
// commit is either a SHA or a pointer (such as HEAD, or HEAD^).
func (g *git) GetDir(commit, dir string) (files []string) {
	output, _ := g.execute("--no-pager", "show", "--name-only", commit+":"+dir)
	parts := strings.SplitN(output, "\n\n", 2) // Split on the blank line
	if len(parts) == 2 && strings.HasPrefix(parts[0], "tree") {
		return strings.Split(strings.TrimRight(parts[1], "\n"), "\n")
	}
	return
}

// SHA retrieves the short form (minimum 8 characters) of the given
// reference.
func (g *git) SHA(ref string) (sha string) {
	commit, _ := g.execute("rev-parse", "--short=8", ref)
	return strings.TrimRight(commit, "\n")
}

// Tags retrieves a list of all tag names from the repository.
func (g *git) Tags() (tags []string) {
	t, _ := g.execute("tag", "--list")
	return strings.Split(strings.TrimRight(t, "\n"), "\n")
}

func (g *git) TotalCommits() (commits int) {
	c, _ := g.execute("rev-list", "--all")
	return len(strings.Split(strings.TrimRight(c, "\n"), "\n"))
}

func (g *git) RefExists(ref string) (exists bool) {
	// If the exit status of 'git rev-list -n 1 <ref>' is nonzero, the
	// ref does not exist in the current repository.
	_, err := g.execute("rev-list", "-n 1", ref)
	return err == nil
}

// Commits parses the log and returns an array of Commit types, up to
// the given max.
func (g *git) Commits(ref string, max int) (commits []*Commit) {
	return g.parseLog(ref, max)
}

// CommitsByFile retrieves a list of commits which modify or otherwise
// affect a file, up to the given maximum number of commits.
func (g *git) CommitsByFile(ref, file string, max int) (commits []*Commit) {
	return g.parseLog(ref, max, "--follow", "--", file)
}

// parseLog is a low-level utility for calling `git log` and producing
// a []*Commit with no phantom commits. It invokes gitParseCommit to
// parse individual commits.
func (g *git) parseLog(ref string, max int, arguments ...string) (commits []*Commit) {
	// First, we have to go through the arduous process of creating
	// the command.
	command := []string{"--no-pager", "log", ref,
		"--format=format:" + gitLogFmt + gitLogSep}
	if max > 0 {
		command = append(command, "-n "+strconv.Itoa(max))
	}
	command = append(command, arguments...)

	log, _ := g.execute(command...)
	// Now we must parse the output of that command.
	commitLogs := strings.Split(log, gitLogSep)
	// We will have a phantom commit here, though, so we must remove
	// it.
	commitLogs = commitLogs[:len(commitLogs)-1]

	commits = make([]*Commit, len(commitLogs))
	for n, l := range commitLogs {
		commits[n] = gitParseCommit(strings.Split(l, "\n"))
	}
	return
}

// gitParseCommit is a low-level utility for parsing log formats of
// the following format. They are generated like this by gitLogFmt.
//    <full hash>
//    <commit time relative>
//    <author name>
//    <nonwrapped commit message>
func gitParseCommit(log []string) (commit *Commit) {
	commit = new(Commit)
	for _, l := range log {
		if len(commit.SHA) == 0 {
			// If l is empty, then this will be run again.
			commit.SHA = l
			continue
		}
		if len(commit.Time) == 0 {
			commit.Time = l
			continue
		}
		if len(commit.Author) == 0 {
			commit.Author = l
			continue
		}
		if len(commit.Email) == 0 {
			commit.Email = l
			continue
		}
		if len(commit.Subject) == 0 {
			commit.Subject = l
			continue
		}

		commit.Body += l + "\n"
	}

	// Now, remove the trailing "\n" characters.
	commit.Body = strings.TrimRight(commit.Body, "\n")
	return
}

// execute invokes exec.Command() with the given command, arguments,
// and working directory. All CR ('\r') characters are removed in
// output.
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
