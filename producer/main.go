package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/streadway/amqp"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}

func bodyFrom(args []string) (string, error) {
	var s string
	if len(args) == 3 {
		s = strings.Join(args[1:], " ")
	} else if len(args) < 2 || os.Args[1] == "" {
		s = "https://en.wikipedia.org/wiki/Computer_science https://en.wikipedia.org/wiki/Manga"
	} else {
		return "", errors.New("incorrect input! Two wiki links are needed")
	}
	return s, nil
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/%2f")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"work_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to declare a queue")

	body, err := bodyFrom(os.Args)
	failOnError(err, "Failed to read args")

	err = ch.Publish(
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "text/plain",
			Body:         []byte(body),
		},
	)
	failOnError(err, "Failed to publish a message")
	log.Printf(" [x] Sent %s", body)
}
