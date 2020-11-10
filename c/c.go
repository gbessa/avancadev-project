package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type Result struct {
	Status string
}

func main() {
	http.HandleFunc("/", process)
	http.ListenAndServe(":9092", nil)
}

func process(w http.ResponseWriter, r *http.Request) {

	ccNumber := r.FormValue("ccNumber")

	valid := check(ccNumber)

	result := Result{Status: valid}

	jsonResult, err := json.Marshal(result)
	if err != nil {
		log.Fatal("Error converting json")
	}

	fmt.Fprintf(w, string(jsonResult))

}

func check(ccNumber string) string {
	if len(ccNumber) == 4 {
		return "valid"
	}
	return "invalid"
}
