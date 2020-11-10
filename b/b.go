package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

type Result struct {
	Status string
}

type Order struct {
	ID        uuid.UUID
	FirstName string
	CcNumber  string
}

func NewOrder() Order {
	return Order{ID: uuid.NewV4()}
}

const INVALIDCC = "invalid"
const VALIDCC = "valid"
const CONNECTION_ERROR = "connection error"

func main() {
	//messageChannel := make(chan amqp.Delivery)

	conn, err := amqp.Dial("amqp://rabbitmq:rabbitmq@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	msgs, err := ch.Consume(
		"orders",         // queue
		"microservice-b", // consumer
		false,            // auto-ack
		false,            // exclusive
		false,            // no-local
		false,            // no-wait
		nil,              // args
	)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for msg := range msgs {
			log.Printf("Received a message: %s", msg.Body)
			processMsgFromQueue(msg)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever

}

func processMsgFromQueue(msg amqp.Delivery) {

	order := NewOrder()
	json.Unmarshal(msg.Body, &order)

	resultValidCC := makeHttpCall("http://localhost:9092", order.CcNumber)
	log.Println(resultValidCC)

	switch resultValidCC.Status {
	case CONNECTION_ERROR:
		msg.Reject(false) //Se não conseguiu falar com o MS3, não deixa tirar a MSG da fila
		log.Println("Order: ", order.ID, ": could not be processed!")
	case INVALIDCC:
		msg.Ack(true)
		log.Println("Order: ", order.ID, ": invalid credit card number! ", order.CcNumber)
	case VALIDCC:
		msg.Ack(true)
		log.Println("Order: ", order.ID, ": processed!")
	default:
		log.Println("Unsupported type")
	}

}

func makeHttpCall(urlMicroService string, ccNumber string) Result {
	values := url.Values{}
	values.Add("ccNumber", ccNumber)

	res, err := http.PostForm(urlMicroService, values)
	if err != nil {
		log.Println("Connection error B to C")
		result := Result{Status: CONNECTION_ERROR}
		return result
	}
	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Println("Connection error B to C 2")
		result := Result{Status: CONNECTION_ERROR}
		return result
	}

	result := Result{}

	json.Unmarshal(data, &result)

	return result

}
