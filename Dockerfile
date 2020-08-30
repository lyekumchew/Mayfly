FROM golang:alpine as build
ADD . src
WORKDIR src
RUN GOOS=linux CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /src/mayfly main.go

FROM alpine
COPY --from=build /src/mayfly /mayfly

EXPOSE 6060

ENV ENDPOINT=
ENV ACCESSKEYID=
ENV SECRETACCESSKEY=
ENV BUCKETNAME=
ENV BASE=
ENV SECRETACCESSKEY=
ENV LIMIT="128M"
ENV ADDR=":6060"

ENTRYPOINT ["/bin/sh", "-c", "exec /mayfly -endpoint ${ENDPOINT} -accessKeyID ${ACCESSKEYID} \
-secretAccessKey ${SECRETACCESSKEY} -bucketName ${BUCKETNAME} -base ${BASE} -limit ${LIMIT} -addr ${ADDR}"]