build-leash:
	docker build -f src/leash/Dockerfile . -t herohamp/mkrcx-leash

build-leash-amd:
	docker buildx build -f src/leash/Dockerfile . -t herohamp/mkrcx-leash --platform linux/amd64