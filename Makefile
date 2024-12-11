build:
	cd face && make build
	mkdir -p ass/dist/assets
	mv face/dist/index.html ass/dist/index.html
	mv face/dist/assets/* ass/dist/assets/
	docker compose up --build -d

.PHONY: build
