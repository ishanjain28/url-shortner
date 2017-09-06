FROM ubuntu
RUN apt-get update
RUN apt-get -y install sqlite3
RUN apt-get -y install curl 

COPY url-shortner /usr/bin
ENV HOST fcc-shorten-urls.herokuapp.com

CMD /usr/bin/url-shortner
