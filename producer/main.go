package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
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

func randomString(len int) string {
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		bytes[i] = byte(rand.Intn(90-65) + 65)
	}
	return string(bytes)
}

func findDistanceRPC(links string) (res []string, err error) {
	conn, err := amqp.Dial("amqp://" + *rabbitAddr)
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

	msgs, err := ch.Consume(
		q.Name,
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
		q.Name,
		false,
		false,
		amqp.Publishing{
			ContentType:   "text/plain",
			CorrelationId: corrId,
			ReplyTo:       q.Name,
			Body:          []byte(links),
		},
	)
	failOnError(err, "Failed to publish a message")
	log.Printf(" [x] Sent %s", links)

	for d := range msgs {
		if corrId == d.CorrelationId {
			str := string(d.Body)

			if str == "" {
				err = errors.New("something went wrong :(")
			} else {
				err = nil
				res = strings.Fields(str)
			}

			break
		}
	}

	return
}

func main() {
	rand.Seed(time.Now().UnixNano())

	body, err := bodyFrom(os.Args)
	failOnError(err, "Failed to read args")

	path, err := findDistanceRPC(body)
	failOnError(err, "Failed to calculate distance")

	if err == nil {
		log.Printf(" [.] Got answer: %d\n     %s", len(path)-1, strings.Join(path, " -> "))
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
