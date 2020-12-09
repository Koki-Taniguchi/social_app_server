.PHONY: restart
restart: stop run-all

.PHONY: run-all
run-all: build run

.PHONY: build
build: image-build pack-build

.PHONY: image-build
image-build:
	docker build -t $(IMAGE_NAME) -f Dockerfile .

.PHONY: pack-build
pack-build:
	pack build \
		--builder $(IMAGE_NAME) \
		--env GOOGLE_FUNCTION_SIGNATURE_TYPE=http \
		--env GOOGLE_FUNCTION_TARGET=HTTPFunction \
		$(IMAGE_NAME)

.PHONY: run
run:
	docker run -d --name $(IMAGE_NAME) --rm \
	-e GOOGLE_CLOUD_PROJECT \
	-e GOOGLE_APPLICATION_CREDENTIALS \
	-e BUCKET_NAME \
	-p 8080:8080 \
	$(IMAGE_NAME)

.PHONY: stop
stop:
	docker stop $(IMAGE_NAME)

.PHONY: ngrok
ngrok:
	ngrok http 8080
