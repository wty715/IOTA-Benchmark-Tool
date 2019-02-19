package transactions

import (
	"fmt"
	"github.com/pebbe/zmq4"
	"strings"
	"strconv"
	"time"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

var MsMsgReceived = 0
var TxMsgReceived = 0
var ConfirmedMsgReceived = 0

type Transaction struct {
	Hash         string `json:"hash"`
	Address      string `json:"address"`
	Value        int    `json:"value"`
	ObsoleteTag  string `json:"obsolete_tag"`
	Timestamp    int64  `json:"timestamp"`
	CurrentIndex int    `json:"current_index"`
	LastIndex    int    `json:"last_index"`
	BundleHash   string `json:"bundle_hash"`
	TrunkTxHash  string `json:"trunk_tx_hash"`
	BranchTxHash string `json:"branch_tx_hash"`
	ArrivalTime  int64  `json:"arrival_time"`
	Tag          string `json:"tag"`
}

var buckets = map[string]*Bucket{}

type Bucket struct {
	TXs []*Transaction
}

var	Inherent_lat	= map[string]int64
var Confirming_lat	= map[string]int64
var Latency			= map[string]int64

func (b *Bucket) full() bool {
	size := len(b.TXs)
	return size != 0 && size == b.TXs[0].LastIndex+1
}

var Start_time int64 = time.Now().Unix()

func StartLog() {
	for {
		lastTotalTxs := TxMsgReceived
		lastTime := time.Now().Unix()
		Confirming_lat = make(map[string]int64)
		Latency = make(map[string]int64)

		time.After(time.Duration(120) * time.Second)

		var totalLatency		int64 = 0
		var totalInherent_lat	int64 = 0
		var totalConforming_lat	int64 = 0
		var total = 0
		for k, v := range Latency {
			totalLatency += v
			totalInherent_lat += Inherent_lat[k]
			totalConforming_lat += Confirming_lat[k]
			total++
		}

		a := time.Now().Unix() - lastTime
		b := time.Now().Unix() - Start_time
		
		fmt.Printf("[%d s - %d s]: Average Latency %f,\n", a, b, totalLatency/total)
		fmt.Printf("[%d s - %d s]: Including inherent latency %f and confirming latency %f.\n", a, b, totalInherent_lat/total, totalConforming_lat/total)
		fmt.Printf("[%d s - %d s]: Average Throughput %d TPS.\n", a, b, (TxMsgReceived-lastTotalTxs)/120)
	}
}

func StartTxFeed(address string) {
	socket, err := zmq4.NewSocket(zmq4.SUB)
	must(err)
	socket.SetSubscribe("tx")
	err = socket.Connect(address)
	must(err)

	fmt.Printf("started tx feed\n")
	for {
		msg, err := socket.Recv(0)
		must(err)
		tx := buildTxFromZMQData(msg)
		if tx == nil {
			//fmt.Printf("receive error! no transaction message\n")
			continue
		}
		var has bool

		// calculate inherent latency
		_, has = Inherent_lat[tx.Hash]
		if !has {
			if tx.ArrivalTime - tx.Timestamp > 0 {
				Inherent_lat[tx.Hash] = tx.ArrivalTime - tx.Timestamp
			}
		} else {
			//fmt.Printf("error! transanction repeated\n")
		}

		// add transaction to bucket
		var b *Bucket
		b, has = buckets[tx.BundleHash]
		if !has {
			b = &Bucket{TXs: []*Transaction{}}
			b.TXs = append(b.TXs, tx)
			buckets[tx.BundleHash] = b
		} else {
			b.TXs = append(b.TXs, tx)
		}
		//fmt.Printf("new transaction attached: %+v\n", tx)
		if b.full() {
			//fmt.Printf("new bundle bucket complete: %+v\n", b)
		}
		TxMsgReceived++
	}
}

type Milestone struct {
	Hash string `json:"hash"`
}

func StartMilestoneFeed(address string) {
	socket, err := zmq4.NewSocket(zmq4.SUB)
	must(err)
	socket.SetSubscribe("lmhs")
	err = socket.Connect(address)
	must(err)

	fmt.Printf("started milestone feed\n")
	for {
		msg, err := socket.Recv(0)
		must(err)
		msgSplit := strings.Split(msg, " ")
		if len(msgSplit) != 2 {
			//fmt.Printf("receive error! milestone message format error\n")
			continue
		}
		MsMsgReceived++
		milestone := Milestone{msgSplit[1]}
		//fmt.Printf("new milestone attached: %+v\n", msg)
		//fmt.Printf("new milestone attached hash: %+v\n", milestone)
	}
}

type ConfTx struct {
	Hash string `json:"hash"`
}

func StartConfirmationFeed(address string) {
	socket, err := zmq4.NewSocket(zmq4.SUB)
	must(err)
	socket.SetSubscribe("sn")
	err = socket.Connect(address)
	must(err)

	fmt.Printf("started confirmation feed\n")
	for {
		msg, err := socket.Recv(0)
		must(err)
		msgSplit := strings.Split(msg, " ")
		if len(msgSplit) != 7 {
			//fmt.Printf("receive error! confirm message format error\n")
			continue
		}
		ConfirmedMsgReceived++
		confTx := ConfTx{msgSplit[2]}

		_, has = Confirming_lat[confTx.Hash]
		if !has {
			if tx.ArrivalTime - tx.Timestamp > 0 {
				Confirming_lat[confTx.Hash] = time.Now().Unix() - confTx.ArrivalTime
				Latency[tx.Hash] = Inherent_lat[tx.Hash] + Confirming_lat[tx.Hash]
			}
		} else {
			//fmt.Printf("error! transanction repeated\n")
		}
		//fmt.Printf("confirm transaction: %+v\n", msg)
		//fmt.Printf("confirm transaction hash: %+v\n", confTx)
	}
}

func buildTxFromZMQData(msg string) *Transaction {
	msgSplit := strings.Split(msg, " ")
	if len(msgSplit) != 13 {
		return nil
	}
	var err error
	msgSplit = msgSplit[1:]
	tx := &Transaction{}
	tx.Hash = msgSplit[0]
	tx.Address = msgSplit[1]
	tx.Value, err = strconv.Atoi(msgSplit[2])
	if err != nil {
		return nil
	}
	tx.ObsoleteTag = msgSplit[3]
	tx.Timestamp, err = strconv.ParseInt(msgSplit[4], 10, 64)
	if err != nil {
		return nil
	}
	tx.CurrentIndex, err = strconv.Atoi(msgSplit[5])
	if err != nil {
		return nil
	}
	tx.LastIndex, err = strconv.Atoi(msgSplit[6])
	if err != nil {
		return nil
	}
	tx.BundleHash = msgSplit[7]
	tx.TrunkTxHash = msgSplit[8]
	tx.BranchTxHash = msgSplit[9]
	tx.ArrivalTime, err = strconv.ParseInt(msgSplit[10], 10, 64)
	if err != nil {
		return nil
	}
	tx.Tag = msgSplit[11]
	return tx
}
