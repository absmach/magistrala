# Contributing to Mainflux

The following is a set of guidelines to contribute to Mainflux and its libraries, which are
hosted on the [Mainflux Organization](https://github.com/mainflux) on GitHub.

This project adheres to the [Contributor Covenant 1.2](http://contributor-covenant.org/version/1/2/0).
By participating, you are expected to uphold this code. Please report unacceptable behavior to
[abuse@mainflux.com](mailto:abuse@mainflux.com).

## Reporting issues

Reporting issues are a great way to contribute to the project. We are perpetually grateful about a well-written,
thorough bug report.

Before raising a new issue, check [our issue
list](https://github.com/mainflux/mainflux/issues) to determine if it already contains the
problem that you are facing.

A good bug report shouldn't leave others needing to chase you for more information. Please be as detailed as possible. The following questions might serve as a template for writing a detailed
report:

- What were you trying to achieve?
- What are the expected results?
- What are the received results?
- What are the steps to reproduce the issue?
- In what environment did you encounter the issue?

## Pull requests

Good pull requests (e.g. patches, improvements, new features) are a fantastic help. They should
remain focused in scope and avoid unrelated commits.

**Please ask first** before embarking on any significant pull request (e.g. implementing new features,
refactoring code etc.), otherwise you risk spending a lot of time working on something that the
maintainers might not want to merge into the project.

Please adhere to the coding conventions used throughout the project. If in doubt, consult the
[Effective Go](https://golang.org/doc/effective_go.html) style guide.

To contribute to the project, [fork](https://help.github.com/articles/fork-a-repo/) it,
clone your fork repository, and configure the remotes:

```
git clone https://github.com/<your-username>/mainflux.git
cd mainflux
git remote add upstream https://github.com/mainflux/mainflux.git
```

If your cloned repository is behind the upstream commits, then get the latest changes from upstream:

```
git checkout master
git pull --rebase upstream master
```

Create a new topic branch from `master` using the naming convention `MF-[issue-number]`
to help us keep track of your contribution scope:

```
git checkout -b MF-[issue-number]
```

Commit your changes in logical chunks. When you are ready to commit, make sure
to write a Good Commit Message™. Consult the [Erlang's contributing guide](https://github.com/erlang/otp/wiki/Writing-good-commit-messages)
if you're unsure of what constitutes a Good Commit Message™. Use [interactive rebase](https://help.github.com/articles/about-git-rebase)
to group your commits into logical units of work before making it public.

Note that every commit you make must be signed. By signing off your work you indicate that you
are accepting the [Developer Certificate of Origin](https://developercertificate.org/).

Use your real name (sorry, no pseudonyms or anonymous contributions). If you set your `user.name`
and `user.email` git configs, you can sign your commit automatically with `git commit -s`.

Locally merge (or rebase) the upstream development branch into your topic branch:

```
git pull --rebase upstream master
```

Push your topic branch up to your fork:

```
git push origin MF-[issue-number]
```

[Open a Pull Request](https://help.github.com/articles/using-pull-requests/) with a clear title
and detailed description.
