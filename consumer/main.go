package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/streadway/amqp"
)

var (
	rabbitAddr = flag.String("addr", "guest:guest@localhost:5672", "RabbitMQ address in the format login:password@host:port")
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func main() {
	flag.Parse()

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

	msgs, err := ch.Consume(
		wq.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf(" [+] Received a message: %s", d.Body)

			timer := time.Now()

			links := strings.Fields(string(d.Body))

			path, err := findPath(links[0], links[1])
			if err != nil {
				log.Printf("Failed to find path: %s", err)
			} else {
				log.Printf(" [.] Got answer: %d\n\t%s", len(path)-1, strings.Join(path, " -> "))
			}
			body := strings.Join(path, " ")

			err = ch.Publish(
				"",
				rq.Name,
				false,
				false,
				amqp.Publishing{
					DeliveryMode:  amqp.Persistent,
					ContentType:   "text/plain",
					CorrelationId: d.CorrelationId,
					Body:          []byte(body),
				},
			)
			failOnError(err, "Failed to publish a message")
			log.Printf(" [.] Sent: %s", body)

			log.Printf(" [-] Done with %f s\n\n", time.Since(timer).Seconds())
		}
	}()

	log.Print(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
