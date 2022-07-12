#!/bin/bash

docker exec -it mysql_backend.admin_panel /etc/init.d/mysql stop
docker exec -it mysql_backend.admin_panel /usr/bin/mysql_install_db
docker exec -it mysql_backend.admin_panel /etc/init.d/mysql start
docker exec -it mysql_backend.admin_panel /usr/bin/mysqladmin -u root password '123456'
docker exec -it mysql_backend.admin_panel mysql -u root -p123456 -e "CREATE DATABASE goshop CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
docker exec -it mysql_backend.admin_panel mysql -u root -p123456 -e "CREATE USER 'goshop'@'%' IDENTIFIED BY '123456'"
docker exec -it mysql_backend.admin_panel mysql -u root -p123456 -e "GRANT ALL PRIVILEGES ON goshop . * TO 'goshop'@'%'"
docker exec -it mysql_backend.admin_panel mysql -u root -p123456 -e "FLUSH PRIVILEGES"

docker exec -it mysql_backend.admin_panel mysql -u root -p123456 -e "create table goshop.pages(id int not null primary key, name varchar(50));"
docker exec -it mysql_backend.admin_panel mysql -u root -p123456 -e "insert into goshop.pages(id, name) values(1, 'test1');"
docker exec -it mysql_backend.admin_panel mysql -u root -p123456 -e "insert into goshop.pages(id, name) values(2, 'test2');"
docker exec -it mysql_backend.admin_panel mysql -u root -p123456 -e "insert into goshop.pages(id, name) values(3, 'test3');"
INSERT INTO keywords SET keyword_name='{	планкен	}';