services-start:
	docker-compose -f docker/docker-compose.yml up -d

services-stop:
	docker-compose -f docker/docker-compose.yml down

mysql-init:
	docker/mysql/init.sh

build:
	go build -o bin/service cmd/service/main.go

run:
	./bin/service

start:
	go build -o bin/service cmd/service/main.go
	./bin/service

test:
	curl http://localhost:8080/admin/page/ -d ''
