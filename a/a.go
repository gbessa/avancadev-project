package main

import (
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/streadway/amqp"
)

type Result struct {
	Status string
}

type Order struct {
	FirstName string
	CcNumber  string
}

func main() {
	http.HandleFunc("/", home)
	http.HandleFunc("/process", process)
	http.ListenAndServe(":9090", nil)
}

func home(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.ParseFiles("templates/home.html"))
	t.Execute(w, Result{})
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func process(w http.ResponseWriter, r *http.Request) {

	firstName := r.FormValue("firstName")
	ccNumber := r.FormValue("cc-number")

	order := Order{
		FirstName: firstName,
		CcNumber:  ccNumber,
	}

	jsonOrder, err := json.Marshal(order)
	if err != nil {
		log.Fatal("Erro na converção da Order em Json")
	}

	conn, err := amqp.Dial("amqp://rabbitmq:rabbitmq@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	body := jsonOrder

	err = ch.Publish(
		"orders_ex", // exchange
		"",          // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(body),
		})
	failOnError(err, "Failed to publish a message!")

	log.Println("Message sent to Queue!!")
	t := template.Must(template.ParseFiles("templates/process.html"))
	t.Execute(w, true)

}

func makeHttpCall(urlMicroService string, firstName string, ccNumber string) Result {
	values := url.Values{}
	values.Add("firstName", firstName)
	values.Add("ccNumber", ccNumber)

	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 5

	res, err := retryClient.PostForm(urlMicroService, values)

	if err != nil {
		//log.Fatal("Microservice B out")
		result := Result{Status: "Servidor foda do ar!"}
		return result
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		//log.Fatal("Error processing result")
		result := Result{Status: "Servidor foda do ar!"}
		return result
	}

	result := Result{}

	json.Unmarshal(data, &result)

	return result

}
