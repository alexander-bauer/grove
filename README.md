<p align="center">![Logo by Dansup, many many thanks :D](https://raw.github.com/SashaCrofter/grove/development/res/logo.png)<br><br>
<img src="http://i.imgur.com/z5Te9.png" /></p><br><br>
<b>Grove is a git hosting application that allows developers to share their local repositories directly with other developers, without needing to push them to a central server.</b> This is all accomplished through a basic web interface, and invocations of the git-http-backend, which comes prepackaged in git distributions.

This use of the very efficient git http capabilities allows developers to utilize the true peer to peer abilities of git, and share cutting-edge changes long even before they've reached the main server.

This sort of workflow, in which project members collaborate directly with other members, is [encouraged with git](http://git-scm.com/about/distributed). It ties development groups much closer together, and gives all members access to the most absolutely recent versions of branches.

##To Install

As Grove is reaching beta, it has become more suitable for general use. It is now packaged with an install script, but bear in mind that Go, the [language](http://golang.org) that Grove is written in, must be installed and configured already. To install Golang, please follow [these instructions](http://golang.org/doc/install).

If Go is already installed (or you've just installed and configured it,) the installation of Grove is as follows.

1. Clone the repository. You may need to run this as root. (Prepend `sudo` to the commands.)
 * via the Go tool
`go get github.com/SashaCrofter/grove` <br>
The repository will be at `$GOPATH/src/github.com/SashaCrofter/grove`
 * via Git
`git clone https://github.com/SashaCrofter/grove.git` <br>
// The repository will be in the current directory under grove/

2. Change directories to the repository.
 * if installed via the Go tool,
`cd $GOPATH/src/github.com/SashaCrofter/grove`
//If `$GOPATH` is not set, replace it with `/usr/local/go`
 * if installed via Git
cd grove
3. Retrieve dependencies. You may need to run this as root.
`go get`
4. Build.
`go build`
5. Install. (This should run as root.)
`sudo ./install.sh skipbuild`


<b>Full instructions for running and administering will come as Grove reaches beta.</b>

##Developer Chat

Join the official development channel for more up-to-date development news and help.
[#grove](http://hypeirc.net) on [Hyperboria](http://hyperboria.net)'s IRC network.

##Active Grove Instances
- [Sasha Crofter](http://[fcdf:db8b:fbf5:d3d7:64a:5aa3:f326:149c]:8860/go/src/github.com/SashaCrofter/grove)
- [Luke Evers](http://[fc2e:9943:1633:403e:2346:3704:8cd8:1c78]:8860/go/src/grove)
- [inhies](http://[fc82:58f9:945f:1b6b:b44:40b:5d89:380f]:8860/)
- [dylwhich](http://[fc8a:9a25:1d90:4677:13ae:9a61:ea8c:66b5]:8860/)
