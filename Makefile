services-start:
	docker-compose -f docker/docker-compose.yml up -d --build

services-stop:
	docker-compose -f docker/docker-compose.yml down

mysql-init:
	docker exec -it mysql_xmlyandex /init.sh

		
