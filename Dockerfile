FROM alpine:latest
EXPOSE 3307 
COPY Dockerfile /Dockerfile
COPY bin/portproxy /portproxy
RUN apk add -U tzdata
RUN apk add ca-certificates
RUN cp /usr/share/zoneinfo/Asia/Tokyo /etc/localtime
WORKDIR /
ENTRYPOINT ["/portproxy"]
