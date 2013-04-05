![Logo by Dansup, many many thanks :D](https://raw.github.com/SashaCrofter/grove/development/res/logo.png)

# Grove
## Git self-hosting for developers
Copyright ⓒ 2013 Alexander Bauer (GPLv3)

Grove is a git hosting application that allows developers to share their local repositories directly with other developers, without needing to push them to a central server. This is all accomplished through a basic web interface, and invocations of the git-http-backend, which comes prepackaged in git distributions.

This use of the very efficient git http capabilities allows developers to utilize the true peer to peer abilities of git, and share cutting-edge changes long even before they've reached the main server.

This sort of workflow, in which project members collaborate directly with other members, is [encouraged with git](http://git-scm.com/about/distributed). It ties development groups much closer together, and gives all members access to the most absolutely recent versions of branches.

## To Install

As Grove is reaching beta, it has become more suitable for general use. It is now packaged with an install script, but bear in mind that Go, the [language](http://golang.org) that Grove is written in, must be installed and configured already. To install Golang, please follow [these instructions](http://golang.org/doc/install).

If Go is already installed (or you've just installed and configured it,) the installation of Grove is as follows.

1. Clone the repository. You may need to run this as root. (Prepend `sudo` to the commands.)
 * via the Go tool `go get github.com/SashaCrofter/grove`
 * or via Git `git clone https://github.com/SashaCrofter/grove.git`

2. Change directories to the repository.
 * if installed via the Go tool `cd $GOPATH/src/github.com/SashaCrofter/grove` (If `$GOPATH` is not set, replace it with `/usr/local/go`)
 * or if installed via Git `cd grove`
3. Retrieve dependencies. You may need to run this as root. `go get`
4. Build. `go build`
5. Install. (This should run as root.) `sudo ./install.sh skipbuild`

The install script will move the Grove executable to `/usr/bin`, its resources to `/usr/local/share`, and its startup script to `/etc/init.d/grove`. To use the startup script:

```bash
# To start Grove
service grove start

# To stop it
service grove stop
# or, if the script fails,
killall grove

# To restart
service grove restart

# To check whether it's running
service grove status

# To start if it's down, and do nothing if it's running
service grove check
```

Grove will *only* allow web access to a directory if it is marked as globally readable and listable. This is file permission `o+rX`, which can be set with `chmod o+rX <directory>` or `chmod -R o+rX <directory>` to set it recursively. Please be careful in setting these permissions if you have any sensitive projects which you would prefer not to share.

Additionally, Grove will *never* serve files from your working directory. In the repository viewer, it will only ever retrieve files and directories through `git`, which means your uncommitted changes are safe from critical eyes.

It is important to note that by default, Grove will attempt to serve your `~/dev` directory. If this is not where your development directory is, you should edit the `/etc/init.d/grove` file to set `DEV` to your desired directory. For example:

```bash
# ...

if [ -z $DEV ]; then
   DEV=~/mycode
fi
# ...
```

Grove will, by default, write logs to `/tmp/grove.log`. This can be set in a similar manner to `DEV`.

Please bear in mind that Grove is beta software, and though functional in theory, may contain bugs, unexpected behavior, and nasal demons.

## Developer Chat

Join the official development channel for more up-to-date development news and help. We're around most of the time, and capable of answering any question related to Grove. (Yes, that is a challenge.)
[#grove](http://hypeirc.net) on [Hyperboria](http://hyperboria.net)'s IRC network.

## Developer Instances (*Grovelinks*)
- [Sasha Crofter](http://[fcdf:db8b:fbf5:d3d7:64a:5aa3:f326:149c]:8860/go/src/github.com/SashaCrofter/grove)
- [Luke Evers](http://[fc2e:9943:1633:403e:2346:3704:8cd8:1c78]:8860/go/src/grove)
- [inhies](http://[fc82:58f9:945f:1b6b:b44:40b:5d89:380f]:8860/)
- [dylwhich](http://[fc8a:9a25:1d90:4677:13ae:9a61:ea8c:66b5]:8860/)
