FROM golang:latest as build

WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ferurl

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/ferurl /app/ferurl
COPY web/ /app/web/

EXPOSE 3000
ENTRYPOINT [ "/app/ferurl" ]
