package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"

	"github.com/streadway/amqp"
)

var (
	source     = flag.String("src", "https://en.wikipedia.org/wiki/Computer_science", "Source wiki page in the format of https://*.wikipedia.org/wiki/*")
	target     = flag.String("tgt", "https://en.wikipedia.org/wiki/Manga", "Target wiki page in the format of https://*.wikipedia.org/wiki/*")
	rabbitAddr = flag.String("addr", "guest:guest@localhost:5672", "RabbitMQ address in the format login:password@host:port")
)

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
		rq.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to register a consumer")

	body := *source + " " + *target

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
			body := string(d.Body)
			log.Printf(" [x] Received: %s", body)

			if body == "" {
				fmt.Println("Something went wrong on remote server :<")

				break
			}

			links := strings.Fields(body)
			fmt.Printf("Path length: %d\n\t%s\n", len(links)-1, strings.Join(links, " -> "))

			break
		}
	}
}
