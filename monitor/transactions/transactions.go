package transactions

import (
	"fmt"
	"github.com/pebbe/zmq4"
	"strings"
	"strconv"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

type MsgType byte

var msMsgReceived = 0
var txMsgReceived = 0
var confirmedMsgReceived = 0

type Msg struct {
	Type MsgType     `json:"type"`
	Obj  interface{} `json:"obj"`
}

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
	ArrivalTime  string `json:"arrival_time"`
	Tag          string `json:"tag"`
}

var buckets = map[string]*Bucket{}

type Bucket struct {
	TXs []*Transaction
}

func (b *Bucket) Full() bool {
	size := len(b.TXs)
	return size != 0 && size == b.TXs[0].LastIndex+1
}

func startTxFeed(address string) {
	socket, err := zmq4.NewSocket(zmq4.SUB)
	must(err)
	socket.SetSubscribe("tx")
	err = socket.Connect(address)
	must(err)

	fmt.Printf("started tx feed")
	for {
		msg, err := socket.Recv(0)
		must(err)
		tx := buildTxFromZMQData(msg)
		if tx == nil {
			fmt.Printf("receive error! no transaction message")
			continue
		}

		// add transaction to bucket
		var b *Bucket
		var has bool
		b, has = buckets[tx.BundleHash]
		if !has {
			b = &Bucket{TXs: []*Transaction{}}
			b.TXs = append(b.TXs, tx)
			buckets[tx.BundleHash] = b
		} else {
			b.TXs = append(b.TXs, tx)
		}
		fmt.Printf("new transaction attached: %+v\n",b)
		if b.Full() {
			fmt.Printf("new bundle bucket complete: %s\n", b.TXs[0].BundleHash)
		}

		txMsgReceived++
	}
}

type Milestone struct {
	Hash string `json:"hash"`
}

func startMilestoneFeed(address string) {
	socket, err := zmq4.NewSocket(zmq4.SUB)
	must(err)
	socket.SetSubscribe("lmhs")
	err = socket.Connect(address)
	must(err)

	fmt.Printf("started milestone feed")
	for {
		msg, err := socket.Recv(0)
		must(err)
		msgSplit := strings.Split(msg, " ")
		if len(msgSplit) != 2 {
			fmt.Printf("receive error! milestone message format error")
			continue
		}
		msMsgReceived++
		milestone := Milestone{msgSplit[1]}
		fmt.Printf("new milestone attached: %+v\n",msg)
		fmt.Printf("new milestone attached hash: %+v\n",milestone)
	}
}

type ConfTx struct {
	Hash string `json:"hash"`
}

func startConfirmationFeed(address string) {
	socket, err := zmq4.NewSocket(zmq4.SUB)
	must(err)
	socket.SetSubscribe("sn")
	err = socket.Connect(address)
	must(err)

	fmt.Printf("started confirmation feed")
	for {
		msg, err := socket.Recv(0)
		must(err)
		msgSplit := strings.Split(msg, " ")
		if len(msgSplit) != 7 {
			fmt.Printf("receive error! confirm message format error")
			continue
		}
		confirmedMsgReceived++
		confTx := ConfTx{msgSplit[2]}
		fmt.Printf("confirm transaction: %+v\n",msg)
		fmt.Printf("confirm transaction hash: %+v\n",confTx)
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
	tx.ArrivalTime = msgSplit[10]
	tx.Tag = msgSplit[11]
	return tx
}
