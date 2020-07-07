package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

const MaxClients = 100
const TimeoutIncoming = 10 * time.Second
const TimeoutGetUrl = time.Second
const MaxUrlsInQuery = 20
const MaxWorker = 4
const Address = ":6969"

type MyHandler struct {
	sync.Mutex
	count int
	sem   chan struct{}
}

func main() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		osCall := <-c
		log.Printf("system call:%+v", osCall)
		cancel()
	}()

	serve(ctx)
}

func serve(ctx context.Context) {
	mux := http.NewServeMux()
	handler := &MyHandler{
		sem: make(chan struct{}, MaxClients),
	}

	mux.Handle("/", http.TimeoutHandler(handler, TimeoutIncoming, "Timeout error"))

	srv := &http.Server{
		Addr:    Address,
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	log.Printf("server started")

	<-ctx.Done()

	log.Printf("server stopped")

	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := srv.Shutdown(ctxShutDown); err != nil {
		log.Fatalf("server Shutdown Failed:%+s", err)
	}

	log.Printf("server exited properly")
}
