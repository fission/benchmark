package main

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/gomodule/redigo/redis"
)

// Handler is the entry point for this fission function
func Handler(w http.ResponseWriter, r *http.Request) {

	n := runtime.NumCPU()
	runtime.GOMAXPROCS(n)

	quit := make(chan bool)

	for i := 0; i < n; i++ {
		go func() {
			for {
				fmt.Println(fmt.Sprintf("Printing %d", i))
				select {
				case <-quit:
					return
				default:
				}
			}
		}()
	}

	time.Sleep(200 * time.Millisecond)
	for i := 0; i < n; i++ {
		quit <- true
	}
	ts := time.Now().Format(time.RFC3339)
	msg := fmt.Sprintf("{\" Hello, world! Time_stamp\": \"%s\"}", ts)
	conn, err := redis.Dial("tcp", "redis-single-master.redis:6379")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	_, err = conn.Do("INCR", "produced")
	if err != nil {
		log.Fatal(err)
	}
	w.Write([]byte(msg))
}
