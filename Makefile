build:
	cd face && make build
	docker compose up --build -d

.PHONY: build
