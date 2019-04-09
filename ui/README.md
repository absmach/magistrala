# GUI for Mainflux in Elm
Dashboard made with [elm-bootstrap](http://elm-bootstrap.info/).

## Install

### Docker container GUI build

Install [Docker](https://docs.docker.com/install/) and [Docker
compose](https://docs.docker.com/compose/install/), `cd` to Mainflux root
directory and then

`docker-compose -f docker/docker-compose.yml up`

if you want to launch a whole Mainflux docker composition, or just

`docker-compose -f docker/docker-compose.yml up ui`

if you want to launch just GUI.

### Native GUI build

Install [Elm](https://guide.elm-lang.org/install.html) and then run the
following commands:

```
git clone https://github.com/mainflux/mainflux
cd mainflux/ui
make
```

This will produce `index.html` in the _ui_ directory. Still in the _mainflux/ui_
folder, enter

`make run`

and follow the instructions on screen.

**NB:** `make` does `elm make src/Main.elm` and `make run` executes `elm
reactor`. Cf. _Makefile_ for more options.

## Configuration

Open the _src/Env.elm_ file and edit the values of the `env` record.

## Contribute to the GUI development

Follow the instructions above to install and run GUI as a native build. In
_src/Env.elm_ change a `url` field value of the `elm` record to
`http://localhost:80/` (trailing slash `/` is mandatory). Instead of `make run`
you can install [elm-live](https://github.com/wking-io/elm-live) and execute
`elm-live src/Main.elm` to get a live reload when your `.Elm` files change.

Launch Mainflux without ui service, either natively or as a Docker composition.
If you have already launched Mainflux as a Docker composition, simply `cd` to
Mainflux folder and run `docker-compose -f docker/docker-compose.yml stop ui`.
Please follow the [guidelines for Mainflux
contributors](https://mainflux.readthedocs.io/en/latest/CONTRIBUTING/).
