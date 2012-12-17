![Logo by Dansup, many many thanks :D](https://raw.github.com/SashaCrofter/grove/development/res/logo.png)

Grove is a git hosting application that allows developers to share their local repositories directly with other developers, without needing to push them to a central server. This is all accomplished through a basic web interface, and invocations of the git-http-backend, which comes prepackaged in git distributions.

This use of the very efficient git http capabilities allows developers to utilize the true peer to peer abilities of git, and share cutting-edge changes long even before they've reached the main server.

This sort of workflow, in which project members collaborate directly with other members, is one of many encouraged on [git-scm.com](http://git-scm.com/about/distributed). It ties development groups much closer together, and gives all members access to the most absolutely recent versions of branches.

## To Install
*Note: Grove is in alpha. It has (since v0.3.2) succeeded in its basic goal: to allow developers to share git repositories via http. Despite that, it is likely not suitable for general use.*

Grove is written in [golang](http://golang.org). This means that, if you already use Go and have it configured, you can simply type `go install github.com/SashaCrofter/grove`, and it will be downloaded, built, and placed in `$GOPATH/bin`.

If you do not have Go installed, you could either [install and configure it](http://golang.org/doc/install), or download a prebuilt binary. If there is none available, or none recent enough, you may request one, either by [filing an issue](https://github.com/SashaCrofter/grove/issues) or contacting us another way. We'd love to make you one. After all, it means that more people are using our software!

## To Use
Currently, Grove runs perfectly fine from within its git repository. If you clone it from one of our git instances, and build it directly in the repository, it will run without a problem. Bear in mind, though, that if it is moved, its visual elements will break.

To specify a directory to display, just run Grove with an argument. For example:
```
grove ~/dev/
```
This will start Grove by pointing it at `dev/` in your home directory. Thus, you can visit [localhost:8860](http://localhost:8860/) and see your projects.

Grove will only display directories and projects if they are **globally readable**. They must have at least file mode `o+rX`. This can be set with `chmod -R o+rX <dir>`, but *please* be careful.

Full instructions for running and administering will come as Grove nears a more finished state.

#### Grovelinks
- [Sasha Crofter](http://[fcdf:db8b:fbf5:d3d7:64a:5aa3:f326:149c]:8860/go/src/github.com/SashaCrofter/grove)
- [Luke Evers](http://[fc2e:9943:1633:403e:2346:3704:8cd8:1c78]:8860/go/src/grove)
