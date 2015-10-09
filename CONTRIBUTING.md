# Contributing to Mainflux

Thank you for your interest in Mainflux and wish to contribute!

The following is a set of guidelines for contributing to Mainflux and its libraries,
which are hosted in the [Mainflux Organization](https://github.com/Mainflux) on GitHub.
These are just guidelines, not rules, use your best judgment and feel free to
propose changes to this document in a pull request.

This project adheres to the [Contributor Covenant 1.2](http://contributor-covenant.org/version/1/2/0).
By participating, you are expected to uphold this code. Please report unacceptable behavior to [abuse@mainflux.com](mailto:abuse@mainflux.com).

#### Table Of Contents

* [Submitting Issues](#submitting-issues)
* [Pull Requests](#pull-requests)
* [Merge Approval](#merge-approval)
* [Documentation Styleguide](#documentation-styleguide)

## Submitting Issues

A bug is a demonstrable problem that is caused by the code in the repository. Good bug reports are extremely helpful - thank you!

Guidelines for bug reports:

 - Use the GitHub issue search — check if the issue has already been reported.
 - Check if the issue has been fixed — try to reproduce it using the latest master or development branch in the repository.
 - Isolate the problem — ideally create a reduced test case and a live example.

A good bug report shouldn't leave others needing to chase you up for more information. Please try to be as detailed as possible in your report. What is your environment? What steps will reproduce the issue? What browser(s) and OS experience the problem? What would you expect to be the outcome? All these details will help people to fix any potential bugs.

Please setup a [profile picture](https://help.github.com/articles/how-do-i-set-up-my-profile-picture)
  to make yourself recognizable and so we can all get to know each other better.

## Pull requests

Good pull requests - patches, improvements, new features - are a fantastic
help. They should remain focused in scope and avoid containing unrelated
commits.

**Please ask first** before embarking on any significant pull request (e.g.
implementing features, refactoring code, porting to a different language),
otherwise you risk spending a lot of time working on something that the
project's developers might not want to merge into the project.

Please adhere to the coding conventions used throughout a project (indentation,
accurate comments, etc.) and any other requirements (such as test coverage).

* Follow the [JavaScript](https://github.com/styleguide/javascript) styleguide
* Follow the [NodeJS](https://github.com/felixge/node-style-guide) styleguide
* Document new code based on the [Documentation Styleguide](#documentation-styleguide)
* End files with a newline
* Place requires in the following order:
    * Built in Node Modules (such as `path`)
    * Built in Mainflux Modules (such as `coreflux`)
    * Local Modules (using relative paths)

Adhering to the following process is the best way to get your work
included in the project:

1. [Fork](https://help.github.com/articles/fork-a-repo/) the project, clone your
   fork, and configure the remotes:

   ```bash
   # Clone your fork of the repo into the current directory
   git clone https://github.com/<your-username>/mainflux.git

   # Navigate to the newly cloned directory
   cd mainflux

   # Assign the original repo to a remote called "upstream"
   git remote add upstream https://github.com/Mainflux/mainflux.git
   ```

2. If you cloned a while ago, get the latest changes from upstream:

   ```bash
   git checkout master
   git pull upstream master
   ```

3. Create a new topic branch (off the main project development branch) to
   contain your feature, change, or fix.

   This separate branch for each changeset you want us to pull in should contain
   either the issue number in the branch name or an indication of what the feature is.

   If you're working on an issue in the [Issues](https://github.com/Mainflux/mainflux/issues) list for the main Mainflux repo, use the naming convention `mainflux-[issue-num]` for your branch name to help us keep track of what your patch actually fixes:

   ```bash
   # List your current branches
   git branch

   # Create a new branch called mainflux-[issue-num]
   git branch mainflux-[issue-num]

   # Switch to the new brach
   git checkout mainflux-[issue-num]
    ```
   or in one go:

   ```bash
   git checkout -b mainflux-[issue-num]
   ```

4. Commit your changes in logical chunks. Please adhere to these [git commit
   message guidelines](http://tbaggery.com/2008/04/19/a-note-about-git-commit-messages.html)
   or your code is unlikely be merged into the main project. Use Git's
   [interactive rebase](https://help.github.com/articles/about-git-rebase/)
   feature to tidy up your commits before making them public.

   Squash your commits into logical units of work using `git rebase -i` and `git push -f`.
   A logical unit of work is a consistent set of patches that should be reviewed together: for example, upgrading the version of a vendored dependency and taking advantage of its now available new feature constitute two separate units of work. Implementing a new function and calling it in another file constitute a single logical unit of work. The very high majority of submissions should have a single commit, so if in doubt: squash down to one.

   After every commit, make sure the test suite passes. Include documentation changes in the same pull request so that a revert would remove all traces of the feature or fix.

5. Locally merge (or rebase) the upstream development branch into your topic branch:

   ```bash
   # Get the latest code from upstream
   git fetch upstream
   
   # Verigy that you are on the master branch
   git checkout master
   
   # Perform the merge
   git merge upstream/master
   ```
   or in one go:
   
   ```bash
   git pull [--rebase] upstream master
   ```
   If there are no conflict, you are all set, if there are some you can resolve them by hand
   (by editing the conflicting files), or with
   
   ```bash
   git mergetool
   ```
   which will fire up your favourite merger to do a 3-ways merge.
   
   3-ways means you will have your local file on your left, the remote file on your right, and the file in the middle
   is the conflicted one, which you need to solve.
   
   A nice 3-ways merger makes this process very easy, and merging could be fun. To see what you have currently
   installed just do `git mergetool`

6. Push your topic branch up to your fork:

   ```bash
   git push origin mainflux-[issue-num]
   ```

7. [Open a Pull Request](https://help.github.com/articles/using-pull-requests/)
    with a clear title and description.

8. Sign off your pull request
 The sign-off is a simple line at the end of the explanation for the patch.

 By signing-off you indicate that you are accepting the Developer Certificate Of Origin. For now, we are using them same DCO as [Linux kernel developers](http://elinux.org/Developer_Certificate_Of_Origin) are using.

 Your signature certifies that you wrote the patch or otherwise have the right to pass it on as an open-source patch.

 You can simply make a comment something like:

 ```bash
 Signed-off-by: John Doe <john.doe@hisdomain.com>
 ```

 Use your real name (sorry, no pseudonyms or anonymous contributions.)

 If you set your `user.name` and `user.email` git configs, you can sign your commit automatically with `git commit -s`.

**IMPORTANT**: By submitting a patch, you agree to allow the project
owners to license your work under the terms of the [Apache License, Version 2.0](LICENSE).

## Merge approval

Mainflux maintainers use LGTM (Looks Good To Me), or sometimes ACK (Acknowledged) or simple "+1",
in comments on the code review to indicate acceptance.

A change requires LGTMs from an absolute majority of the maintainers of each
component affected. For example, if a change affects `docs/` and `app/`, it
needs an absolute majority from the maintainers of `docs/` AND, separately, an
absolute majority of the maintainers of `app/`.

For more details, see the [MAINTAINERS](MAINTAINERS) page.

## Documentation Styleguide

* Use [Markdown](https://daringfireball.net/projects/markdown)
* Use [Doxygen](https://www.stack.nl/~dimitri/doxygen/manual/docblocks.html) documenting styleguide
* Specifficaly, use [JavaDoc](https://en.wikipedia.org/wiki/Javadoc) flavor
* See [Atomthreads](https://github.com/kelvinlawson/atomthreads/blob/master/kernel/atomkernel.c) project for fantastic example of code commenting
