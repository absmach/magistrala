# GUI for Mainflux in Elm
Dashboard made with [elm-bootstrap](http://elm-bootstrap.info/).

## Install

### Install GUI as a part of Mainflux build

Install Elm (https://guide.elm-lang.org/install.html) and then

```
git clone https://github.com/mainflux/mainflux
cd mainflux/ui
make
```

This will produce `index.html` in the _ui_ directory. In order to use it, `cd`
to _ui_ and do

`make run`

### Build a standalone native GUI

Install Elm (https://guide.elm-lang.org/install.html), `cd` to _ui_ and then

`elm make --optimize src/Main.elm`

This will produce `index.html` in the _ui_ directory. In order to use it do

`make run`

### About Elm `make`

`make` does `elm make src/Main.elm`.

`make run` just executes `elm reactor`. You can execute `elm reactor` in other
terminal window and keep it running, and then see changes as you change-compile
in the first window. You can even use something as
[entr](http://eradman.com/entrproject/) to have your source compiled
automatically when you change and save some files.

### Build as a part of Docker composition

Install Docker (https://docs.docker.com/install/) and Docker compose
(https://docs.docker.com/compose/install/), `cd` to Mainflux root directory and
then

`docker-compose -f docker/docker-compose.yml up`

if you want to launch a whole Mainflux docker composition or just

`docker-compose -f docker/docker-compose.yml up ui`

if you want to launch just GUI.

### Contribute to the GUI development

Install GUI as a part of Mainflux build or as a standalone native GUI and run
it. Launch Mainflux without ui service, either natively or as a Docker
composition. Follow the guidelines for Mainflux contributors found here
https://mainflux.readthedocs.io/en/latest/CONTRIBUTING/.
