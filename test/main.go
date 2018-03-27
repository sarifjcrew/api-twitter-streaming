package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	test1 := make(chan int)
	// test2 := make(chan int)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signalChan
		test1 <- 0
	}()

	go func(test1 <-chan int) <-chan int {
		for {
			select {
			case <-test1:
				return test1
			default:
				fmt.Println("sarif")
			}
		}
	}(test1)
	<-test1
	fmt.Println("end")
}

// func publishVotes(votes <-chan string) <-chan struct{} {
// 	stopchan := make(chan struct{}, 1)
// 	pub, _ := nsq.NewProducer("localhost:4150", nsq.NewConfig())
// 	go func() {
// 		for vote := range votes {
// 			pub.Publish("votes", []byte(vote))
// 		}
//
// 		log.Println("Publisher: Stopping")
// 		pub.Stop()
// 		log.Println("Publisher: Stopped")
// 		stopchan <- struct{}{}
// 	}()
//
// 	return stopchan
// }
