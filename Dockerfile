FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/ginlog .

FROM alpine:3.20
RUN adduser -D -u 10001 ginlog && mkdir -p /data && chown ginlog:ginlog /data
WORKDIR /data
COPY --from=build /out/ginlog /usr/local/bin/ginlog
USER ginlog
ENV DB_PATH=/data/ginlog.db
ENV ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["ginlog"]
