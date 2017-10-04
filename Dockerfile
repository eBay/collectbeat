FROM golang:1.8

RUN mkdir /collectbeat
ADD build/collectbeat /collectbeat/

ENTRYPOINT ["/collectbeat/collectbeat", "-e", "-v", "-c", "/etc/collectbeat/collectbeat.yml"]
