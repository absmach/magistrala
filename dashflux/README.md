# Mainflux UI Dashboad

Mainflux UI, dashboard for [Mainflux](https://github.com/mainflux/mainflux) Industrial IoT Messaging and Device Management Server.

> **N.B.** Mainflux UI service is WIP and not suitable for deployment at this moment. You are welcome to contribute and improve it.
> ## Development
>- Follow angular-cli [documentation](https://github.com/angular/angular-cli)
>- Follow [official angular style guide](https://angular.io/styleguide)

## Requirements

You'll need the following software installed to get started.

- [Node](https://nodejs.org/en/) 6  or higher, we recommend current LTS version, together with NPM 3 or higher.
- [Angular-cli](https://github.com/angular/angular-cli) Newest version with Webpack integration
- - Depending on how Node is configured on your machine, you may need to run installation command with `sudo`
- [Git](http://git-scm.com/downloads): Use the installer for your OS.
  - Windows users can also try [Git for Windows](http://git-for-windows.github.io/).
- For local Development with [Mainflux composition](https://github.com/mainflux/mainflux) running locally, [Chrome extension for Cross origin](https://chrome.google.com/webstore/detail/allow-control-allow-origi/nlfbmbojpeacfghkpbjhddihlkkiljbi?utm_source=chrome-app-launcher-info-dialog) is required. Because composition is running on different port then our Angular app, we have cross origin.

## Configuration

Change into the directory.

```bash
cd dashflux
```

Install the dependencies. If you're running Mac OS or Linux, you may need to run `sudo npm install` instead, depending on how your machine is configured.

```bash
npm install
```

Set appropriate endpoint URLs in **environment.ts** (for local development will probably be 0.0.0.0:<_port_>) or **environment.prod.ts** for production.

To start the server, run:

```bash
ng serve
```

This will run and assemble our app.
 **Now go to `localhost:4200` in your browser to see it in action.**

## Deployment

Dashflux is distributed as Docker container. We use nginx to serve dashflux from docker container, supporting environments using docker multi-stage builds.
Dashflux docker image is available on [Dockerhub mainflux/dashflux](https://hub.docker.com/r/mainflux/dashflux/)

If you want to build image locally, you can build image using the **development** environment:

```bash
docker build -f docker/Dockerfile  -t dashflux:dev --build-arg env=dev .
```

Build image using the **production** environment:

```bash
docker build -t dashflux:prod -f ./docker/Dockerfile .
```

**Note:** before running *docker build* command, please make sure appropriate endpoint URLs in *environment.ts* or *environment.prod.ts* are set up.

You can test image running

```bash
docker run -p 80:80 dashflux:dev
```

This will run dashflux in docker container.

Now go to `http://localhost` in your browser to see it in action.
