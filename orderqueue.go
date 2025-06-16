package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func getOrdersFromQueue() ([]Order, error) {
	var orders []Order

	queueURI := os.Getenv("ORDER_QUEUE_URI")
	queueName := os.Getenv("ORDER_QUEUE_NAME")

	if queueURI == "" || queueName == "" {
		return nil, errors.New("ORDER_QUEUE_URI or ORDER_QUEUE_NAME is not set")
	}

	conn, err := amqp.Dial(queueURI)
	if err != nil {
		log.Printf("Failed to connect to RabbitMQ: %s", err)
		return nil, err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Failed to open a channel: %s", err)
		return nil, err
	}
	defer ch.Close()

	msgs, err := ch.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		log.Printf("Failed to register a consumer: %s", err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for {
		select {
		case d := <-msgs:
			log.Printf("Message received: %s", d.Body)

			order, err := unmarshalOrderFromQueue(d.Body)
			if err != nil {
				log.Printf("Failed to unmarshal order: %s", err)
				d.Nack(false, false) // reject message without requeue
				continue
			}

			orders = append(orders, order)
			d.Ack(false) // acknowledge message

		case <-ctx.Done():
			log.Println("Finished consuming or timeout reached.")
			return orders, nil
		}
	}
}

func unmarshalOrderFromQueue(data []byte) (Order, error) {
	var order Order

	err := json.Unmarshal(data, &order)
	if err != nil {
		log.Printf("Failed to unmarshal order: %v\n", err)
		return Order{}, err
	}

	order.OrderID = strconv.Itoa(rand.Intn(100000))
	order.Status = Pending

	return order, nil
}