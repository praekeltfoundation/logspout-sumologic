FROM gliderlabs/logspout:v3.2.4 AS build-env
ENV LOGSPOUT_SUMOLOGIC_VERSION 0.1.0
ENV LOGSPOUT_VERSION 3.2.4

FROM alpine:3.7
RUN apk add --update ca-certificates
COPY --from=build-env /bin/logspout /bin/logspout
ENTRYPOINT ["/bin/logspout"]
