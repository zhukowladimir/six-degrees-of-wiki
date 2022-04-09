package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/streadway/amqp"
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

	err = ch.Qos(1, 0, false)
	failOnError(err, "Failed to set QoS")

	name := randomString(16)

	msgs, err := ch.Consume(
		q.Name,
		name,
		false,
		false,
		false,
		false,
		nil,
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf(" [.] Received a message: %s", d.Body)

			links := strings.Fields(string(d.Body))
			timer := time.Now()
			path, err := findDistance(links[0], links[1])
			if err != nil {
				log.Printf("Failed to find distance: %s", err)
				path = make([]string, 0)
			}

			err = ch.Publish(
				name,
				d.ReplyTo,
				false,
				false,
				amqp.Publishing{
					ContentType:   "text/plain",
					CorrelationId: d.CorrelationId,
					Body:          []byte(strings.Join(path, " ")),
				},
			)
			failOnError(err, "Failed to publish a message")

			log.Println(" [.] Done with", time.Since(timer).Seconds())

			d.Ack(false)
		}
	}()

	log.Print(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
