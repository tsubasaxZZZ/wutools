FROM golang:latest as build
ADD . /kd
WORKDIR /kd
RUN go get github.com/go-ini/ini && \
    go get github.com/go-sql-driver/mysql && \
    go get github.com/tsubasaxZZZ/wutools/common
RUN go build -o kbdownloader


FROM python:3.5

ADD kbdownloader-nginx  /etc/nginx/sites-available/
ADD supervisord-kd.conf /etc/supervisor/conf.d/
ADD . /kd
COPY --from=build /kd/kbdownloader /kd

RUN set -x && \
    apt-get update && \
    apt-get -y install nginx supervisor cron && \
    ln -sf /etc/nginx/sites-available/kbdownloader-nginx /etc/nginx/sites-enabled/ && \
    cd /kd && \
    pip install -r ./requirements.txt && \
    mkdir /var/run/uwsgi /var/log/uwsgi && \
    chown www-data:www-data /var/run/uwsgi /var/log/uwsgi

EXPOSE 8080
CMD ["/usr/bin/supervisord"]

