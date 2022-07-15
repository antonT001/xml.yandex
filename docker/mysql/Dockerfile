FROM debian:8.6

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update
RUN apt-get install -y mysql-server-5.5 tzdata mc

COPY my.cnf /etc/mysql/my.cnf
COPY init.sh .

RUN rm /etc/localtime
RUN ln -s /usr/share/zoneinfo/Europe/Moscow /etc/localtime 

RUN chown -R mysql:mysql /var/lib/mysql

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT /usr/local/bin/entrypoint.sh
