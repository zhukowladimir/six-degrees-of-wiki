package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"

	"github.com/streadway/amqp"
	"github.com/zhukowladimir/six-degrees-of-wiki/pkg/proto/service"
	"google.golang.org/grpc"
)

var (
	port       = flag.Int("port", 8888, "The server port")
	rabbitAddr = flag.String("addr", "guest:guest@localhost:5672", "RabbitMQ address in the format login:password@host:port")
)

type server struct {
	service.UnimplementedServiceServer

	workQ  amqp.Queue
	replyQ amqp.Queue
}

func (s *server) GetPath(_ context.Context, request *service.Request) (*service.Response, error) {
	msgs, err := ch.Consume(
		s.replyQ.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to register a consumer")

	corrId := randomString(32)

	err = ch.Publish(
		"",
		wq.Name,
		false,
		false,
		amqp.Publishing{
			DeliveryMode:  amqp.Persistent,
			ContentType:   "text/plain",
			CorrelationId: corrId,
			ReplyTo:       rq.Name,
			Body:          []byte(body),
		},
	)
	failOnError(err, "Failed to publish a message")
	log.Printf(" [x] Sent: %s", body)

	for d := range msgs {
		if d.CorrelationId == corrId {
			res := string(d.Body)
			log.Printf(" [x] Received: %s", res)
			break
		}
	}
	return nil, errors.New("something went wrong :<")
}

func main() {
	flag.Parse()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	failOnError(err, "Failed to listen port")

	conn, err := amqp.Dial("amqp://" + *rabbitAddr)
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	wq, err := ch.QueueDeclare(
		"work_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a work queue")

	rq, err := ch.QueueDeclare(
		"reply_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a reply queue")

	thisServer := server{workQ: wq, replyQ: rq}
	srv := grpc.NewServer()
	service.RegisterServiceServer(srv, &thisServer)
	log.Printf("Server listening at %v", lis.Addr().String())

	err = srv.Serve(lis)
	failOnError(err, "Failed to serve")

	msgs, err := ch.Consume(
		rq.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to register a consumer")

	corrId := randomString(32)

	err = ch.Publish(
		"",
		wq.Name,
		false,
		false,
		amqp.Publishing{
			DeliveryMode:  amqp.Persistent,
			ContentType:   "text/plain",
			CorrelationId: corrId,
			ReplyTo:       rq.Name,
			Body:          []byte(body),
		},
	)
	failOnError(err, "Failed to publish a message")
	log.Printf(" [x] Sent: %s", body)

	for d := range msgs {
		if d.CorrelationId == corrId {
			res := string(d.Body)
			log.Printf(" [x] Received: %s", res)
			break
		}
	}

}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func randomString(len int) string {
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		bytes[i] = byte(rand.Intn(90-65) + 65)
	}
	return string(bytes)
}
