package main

import (
    "os"
    "os/signal"
    "syscall"
    "time"
    "fmt"
    "IOTA-Benchmark-Tool/monitor/transactions"
    "github.com/pebbe/zmq4"
    "encoding/json"
    "io/ioutil"
)

func Shutdown(timeout time.Duration) {
    select {
    case <-time.After(timeout):
    }
}

type Config struct {
    Zmq_Address string
    Interval    int
}

var totalTime int64 = 0

func main() {
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

    // print out current zmq version
    major, minor, patch := zmq4.Version()
    fmt.Printf("running ZMQ %d.%d.%d\n", major, minor, patch)

    // read config file
    value := Config{}
    fileBytes, err := ioutil.ReadFile("./config.json")
    if err != nil {
        panic(err)
    }
    if err := json.Unmarshal(fileBytes, &value); err != nil {
        panic(err)
    }
    fmt.Printf("ZMQ addr: %s\n", value.Zmq_Address)
    fmt.Printf("Interval: %d\n", value.Interval)

    // start feeds
    go transactions.StartTxFeed(value.Zmq_Address)
    go transactions.StartMilestoneFeed(value.Zmq_Address)
    go transactions.StartConfirmationFeed(value.Zmq_Address)
    go transactions.StartDoubleFeed(value.Zmq_Address)
    go transactions.StartLog(value.Interval)

    select {
    case <-sigs:
        fmt.Printf("total Time: %d\n", time.Now().Unix() - transactions.Start_time)
        fmt.Printf("total Transactions: %d\n", transactions.TxMsgReceived)
        confirm_lat := [value.Interval*3+10]int
        for _, v := range transactions.Transactions {    
            confirm_lat[int32(v.Inherent_lat)]++
        }
        fmt.Printf("Confirm latency distribution:\n-------------------------------------\n")
        for k, v := range confirm_lat {
            fmt.Printf("| %d\ts | %d\ttransactions |\n", k, v)
        }
        fmt.Printf("-------------------------------------\n")
        Shutdown(time.Duration(1500) * time.Millisecond)
    }
}