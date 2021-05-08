package main

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func startServer(ctx context.Context, g *errgroup.Group, srv *http.Server, name string) {
	g.Go(func() error {
		println("start: ", name)
		return srv.ListenAndServe()  // serve会阻塞
	})

	g.Go(func() error {
		<- ctx.Done()
		println("done, shut down http ", name, time.Now().String())
		ctxShutdown, cancel := context.WithTimeout(context.Background(), time.Second * 3)
		defer cancel()
		err := srv.Shutdown(ctxShutdown)
		if err != nil {
			println("error in shutdown", name, err)
		}
		println(name, "shutdown success", time.Now().String())
		return err
	})
}

func main() {
	c, cancel := context.WithCancel(context.Background())
	defer cancel()

	g, ctx := errgroup.WithContext(c)

	handler1 := http.NewServeMux()
	handler1.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		time.Sleep(time.Second * 2)
		fmt.Fprintf(writer, "server 1, %s", time.Now().String())
	})
	server1 := &http.Server{
		Addr:              ":8081",
		Handler:           handler1,
	}

	handler2 := http.NewServeMux()
	handler2.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		fmt.Fprintf(writer, "server 2")
	})
	server2 := &http.Server{
		Addr:              ":8082",
		Handler:           handler2,
	}

	startServer(ctx, g, server1, "server1")
	startServer(ctx, g, server2, "server2")

	g.Go(func() error {
		exitSignals := []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT}
		sig := make(chan os.Signal, len(exitSignals))
		signal.Notify(sig, exitSignals...)
		select {
		case <-sig:
			println("got sig")
			cancel()
			return nil
		case <-ctx.Done():
			close(sig)
			return ctx.Err()
		}
	})

	err := g.Wait()
	fmt.Printf("g.Wait() %v, %T \n", err, err)
	println("The End.")
}
