FROM golang:1.18

WORKDIR /usr/local/app/

COPY . .

RUN ls

RUN go mod init github.com/zhukowladimir/six-degrees-of-wiki/consumer
RUN go mod tidy
RUN go build -o /consumer

CMD ["bash", "-c", "while ! curl -s localhost:15672 > /dev/null; do echo waiting for rabbitmq; sleep 3; done; /consumer"]
