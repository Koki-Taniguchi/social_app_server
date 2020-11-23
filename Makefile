IMAGE_NAME := social_app_server

.PHONY: run-all
run-all: build run

.PHONY: build
build:
	pack build \
		--builder gcr.io/buildpacks/builder:v1 \
		--env GOOGLE_FUNCTION_SIGNATURE_TYPE=http \
		--env GOOGLE_FUNCTION_TARGET=HTTPFunction \
		--env GOOGLE_DEVMODE=true \
		$(IMAGE_NAME)

.PHONY: run
run:
	docker run -d --name $(IMAGE_NAME) --rm -p 8080:8080 $(IMAGE_NAME)

.PHONY: stop
stop:
	docker stop $(IMAGE_NAME)

.PHONY: ngrok
ngrok:
	ngrok http 8080
