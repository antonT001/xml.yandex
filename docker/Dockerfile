FROM debian:8.6

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update
RUN apt-get install -y mysql-server-5.5 tzdata mc

COPY my.cnf /etc/mysql/my.cnf

RUN rm /etc/localtime
RUN ln -s /usr/share/zoneinfo/Europe/Moscow /etc/localtime 

#docker rmi mysql_server


#CREATE DATABASE xml CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
#CREATE USER 'user'@'%' IDENTIFIED BY '123456'
#GRANT ALL PRIVILEGES ON xml . * TO 'user'@'%'
#FLUSH PRIVILEGES

#CREATE TABLE hosts (id int primary key, host_name varchar(50) not null);
#INSERT INTO `hosts` (`host_name`) VALUES ('lswood.ru');




#
#
#
#
#
#
#
#
#

RUN chown -R mysql:mysql /var/lib/mysql

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT /usr/local/bin/entrypoint.sh
