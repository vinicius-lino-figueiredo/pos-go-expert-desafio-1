package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	u          = "http://localhost:8080/cotacao"
	reqTimeout = 300 * time.Millisecond
)

func main() {

	f, err := os.OpenFile("./data.txt", os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal("opening file:", err.Error())
	}

	defer f.Close()

	ctx, cancel := context.WithTimeout(context.Background(), reqTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		log.Fatal("creating request:", err.Error())
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal("doing request:", err.Error())
	}

	defer res.Body.Close()

	var o requestResponse
	if err := json.NewDecoder(res.Body).Decode(&o); err != nil {
		log.Fatal("unmarshalling response:", err.Error())
	}

	if _, err := fmt.Fprintf(f, "Dólar: %s\n", o.USDBRL.Bid); err != nil {
		log.Fatal("writing file:", err.Error())
	}

	log.Println("exiting")

}

type requestResponse struct {
	USDBRL struct {
		Code       string `json:"code"`
		Codein     string `json:"codein"`
		Name       string `json:"name"`
		High       string `json:"high"`
		Low        string `json:"low"`
		VarBid     string `json:"varBid"`
		PctChange  string `json:"pctChange"`
		Bid        string `json:"bid"`
		Ask        string `json:"ask"`
		Timestamp  string `json:"timestamp"`
		CreateDate string `json:"create_date"`
	} `json:"USDBRL"`
}
