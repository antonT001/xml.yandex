run:
	docker run  -it -p 8080:3306 -v /Users/antontrufanov/Documents/stas/xml.yandex/data_base:/var/lib/mysql --name mysql_server mysql_server

build:
	docker build -t mysql_server .

start:
	docker start mysql_server

stop:
	docker stop mysql_server
