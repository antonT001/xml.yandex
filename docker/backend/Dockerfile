FROM ubuntu

RUN apt-get update
RUN apt-get install golang-go mc ca-certificates -y

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT /usr/local/bin/entrypoint.sh
