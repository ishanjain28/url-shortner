FROM ubuntu
RUN apt-get update
RUN apt-get -y install sqlite3


COPY url-shortner /usr/bin

RUN /usr/bin/url-shortner
