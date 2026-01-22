package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	port       = ":8080"
	u          = "https://economia.awesomeapi.com.br/json/last/USD-BRL"
	reqTimeout = 200 * time.Millisecond
	dbTimeout  = 10 * time.Millisecond
)

var db *sql.DB

func main() {

	var err error
	db, err = sql.Open("sqlite3", "./data.db")
	if err != nil {
		log.Fatal("error opening db connection:", err.Error())
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS dollar_values ( value NUMBER, time TIMESTAMP )`)
	if err != nil {
		log.Fatal("error ensuring table:", err.Error())
	}

	ctx, cancel := context.WithCancelCause(context.Background())

	server := http.Server{Addr: port, Handler: http.HandlerFunc(handle)}

	go func() {
		log.Println("starting server on port", port)
		err := server.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			log.Println("error serving:", err.Error())
		}
		cancel(err)
	}()

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT)
	defer stop()

	<-ctx.Done()

	sdCtx, stop := context.WithTimeout(context.Background(), 30*time.Second)
	defer stop()

	if err := server.Shutdown(sdCtx); err != nil {
		log.Println("error shutting down http server:", err.Error())
	}

	log.Println("exiting")

}

func handle(w http.ResponseWriter, r *http.Request) {

	log.Println("receiving request")

	ctx := r.Context()

	reqCtx, cancel := context.WithTimeout(ctx, reqTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, u, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("error creating request:", err.Error())
		return
	}

	log.Println("calling awsome api")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("error doing request:", err.Error())
		return
	}

	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println("error reading response data:", err.Error())
		return
	}

	var o awesomeAPI

	log.Println("unmarshaling")

	err = json.Unmarshal(b, &o)
	if err != nil {
		log.Println("error unmarshaling:", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	log.Println("storing result")

	dbCtx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()
	_, err = db.ExecContext(dbCtx, `insert into dollar_values (value, time) values ($1, CURRENT_TIMESTAMP)`, o.USDBRL.Bid)
	if err != nil {
		log.Println("error inserting:", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(b)
}

type awesomeAPI struct {
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
