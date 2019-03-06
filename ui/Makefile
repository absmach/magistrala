all: dev

.PHONY: all docker

dev:
	elm make src/Main.elm

prod:
	elm make --optimize src/Main.elm

run:
	elm reactor

docker:
	docker build --tag=mainflux/ui -f docker/Dockerfile .

clean:
	rm -f index.html

mrproper: clean
	rm -rf elm-stuff
