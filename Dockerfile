FROM golang:1.24 AS build
RUN apt-get install gcc g++ make git
WORKDIR /go/src/app
COPY ./src ./src
COPY ./go.* ./
COPY ./Makefile ./
RUN apt-get install tzdata
ENV TZ Europe/Moscow

RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
RUN go mod tidy
RUN CGO_ENABLED=1 make build-linux

FROM golang:1.24
RUN apt-get update -y && apt-get install -y ca-certificates
RUN apt-get install -y tzdata
ENV TZ Europe/Moscow
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
WORKDIR /app
COPY --from=build /go/src/app/build /app/bin

EXPOSE 8080
CMD /app/bin/duckdbm