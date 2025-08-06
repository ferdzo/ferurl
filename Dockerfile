FROM golang:latest as build

WORKDIR /app
COPY . .
RUN go build -o main .

FROM scratch
COPY --from=build /app/main /usr/local/bin/main
