FROM golang:1.23 AS build
RUN apt-get install gcc g++ make git
WORKDIR /go/src/app
COPY ./src ./src
COPY ./go.* ./
COPY ./Makefile ./
RUN apt-get install tzdata
ENV TZ Europe/Moscow

RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
RUN go mod tidy
RUN CGO_ENABLED=0 make build

FROM golang:1.23
RUN apt-get install ca-certificates
RUN apt-get install tzdata
ENV TZ Europe/Moscow
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone
WORKDIR /app
COPY --from=build /go/src/app/build /app/bin

EXPOSE 8080
CMD /app/bin/duckdbm