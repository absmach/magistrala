# Mainflux
Mainflux is an open source MIT licensed IoT cloud written in NodeJS

## Run

### Install Node Modules
```bash
    npm install
```

### Run Gulp Task
```bash
    gulp
```

## Docker
### Build image
```bash
    sudo docker build -t=mainflux .
```

## Run image
```bash
    sudo docker run -i -t -d -p 8080:8080 --name=mainflux mainflux`
```

## License 
MIT
