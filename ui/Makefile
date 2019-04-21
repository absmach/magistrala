all: dev

.PHONY: all docker

dev:
	elm make src/Main.elm --output=main.js

prod:
	elm make --optimize src/Main.elm --output=main.js

run:
	elm reactor

docker:
	docker build --tag=mainflux/ui -f docker/Dockerfile .

clean:
	rm -f main.js

mrproper: clean
	rm -rf elm-stuff
