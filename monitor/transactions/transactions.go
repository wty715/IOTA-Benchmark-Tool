package transactions

import (
    "fmt"
    "github.com/pebbe/zmq4"
    "strings"
    "strconv"
    "time"
    "container/list"
)

func must(err error) {
    if err != nil {
        panic(err)
    }
}

var MsMsgReceived = 0
var TxMsgReceived = 0
var TotalTips = 0
var ConfirmedMsgReceived = 0

type Transaction struct {
    Type         string `json:"type"`
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
    Status       string `json:"status"`
    Inherent_lat int64  `json:"inherent_latency"`
    Confirm_lat  int64  `json:"confirm_latency"`
}
type Bucket struct {
    TXs []*Transaction
}
type Double struct {
    TXs     []*Transaction
    visited bool
}

var buckets = map[string]*Bucket{}
var transactions = map[string]*Transaction{}
var doubles = map[string]*Double{}

func (b *Bucket) full() bool {
    size := len(b.TXs)
    return size != 0 && size == b.TXs[0].LastIndex+1
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
            fmt.Printf("tx: receive error! message format error\n")
            continue
        } else if tx.Type == "tx_trytes" {
            //fmt.Printf("tx: trytes received. Skip.\n")
            continue
        }
        // store transaction & calculate inherent latency
        _, has := transactions[tx.Hash]
        if !has {
            if tx.ArrivalTime+50 > tx.Timestamp*1000 {
                tx.Inherent_lat = tx.ArrivalTime+50 - tx.Timestamp*1000
            } else {
                tx.Inherent_lat = tx.ArrivalTime*1000+50 - tx.Timestamp*1000
            }
            tx.Status = "Tips"
            TotalTips++
            // change tips status
            t, has := transactions[tx.BranchTxHash]
            if has {
                if t.Status == "Tips" {
                    TotalTips--
                    t.Status = "Approved"
                }
            }
            t, has = transactions[tx.TrunkTxHash]
            if has {
                if t.Status == "Tips" {
                    TotalTips--
                    t.Status = "Approved"
                }
            }
            // store txs and create graph
            transactions[tx.Hash] = tx
            d, has := doubles[tx.BranchTxHash]
            if !has {
                d = &Double{TXs: []*Transaction{}, visited: false}
                d.TXs = append(d.TXs, tx)
                doubles[tx.BranchTxHash] = d
            } else {
                d.TXs = append(d.TXs, tx)
            }
            d, has = doubles[tx.TrunkTxHash]
            if !has {
                d = &Double{TXs: []*Transaction{}, visited: false}
                d.TXs = append(d.TXs, tx)
                doubles[tx.TrunkTxHash] = d
            } else {
                d.TXs = append(d.TXs, tx)
            }
        } else {
            fmt.Printf("tx: error! transanction repeated\n")
            continue
        }
        // add transaction to bucket
        b, has := buckets[tx.BundleHash]
        if !has {
            b = &Bucket{TXs: []*Transaction{}}
            b.TXs = append(b.TXs, tx)
            buckets[tx.BundleHash] = b
        } else {
            b.TXs = append(b.TXs, tx)
        }
        if b.full() {
            //fmt.Printf("new bundle bucket complete: %+v\n", b)
        }
        TxMsgReceived++
        //fmt.Printf("new transaction attached: %+v\n", tx)
        //fmt.Printf("RAW msg: %s\n", msg)
    }
}

type ConfTx struct {
    Hash         string `json:"hash"`
    Address	     string `json:"address"`
    TrunkTxHash  string `json:"trunk_tx_hash"`
    BranchTxHash string `json:"branch_tx_hash"`
    BundleHash   string `json:"bundle_hash"`
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
        tx := buildConfirmFromZMQData(msg)
        if tx == nil {
            fmt.Printf("confirm: receive error! message format error\n")
            continue
        }
        // calculate confirming latency
        t, has := transactions[tx.Hash]
        if has {
            if t.ArrivalTime+50 > t.Timestamp*1000 {
                t.Confirm_lat = int64(time.Now().UnixNano()/1e6) - t.ArrivalTime
            } else {
                t.Confirm_lat = int64(time.Now().UnixNano()/1e6) - t.ArrivalTime*1000
            }
            if t.Confirm_lat < 0 {
                fmt.Printf("ERROR!!! nowTime %d, Inherent_lat %d\n", int64(time.Now().UnixNano()/1e6), t.Inherent_lat)
                fmt.Printf("ERROR!!! transaction: %+v\n", *t)
            }
            if t.Status == "Tips" {
                TotalTips--
            }
            t.Status = "Confirmed"
            Interval_txs = append(Interval_txs, tx.Hash)
            ConfirmedMsgReceived++
        } else {
            continue
        }
        //fmt.Printf("confirm transaction: %+v\n", tx)
        //fmt.Printf("RAW msg: %s\n", msg)
    }
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
            fmt.Printf("milestone: receive error! message format error\n")
            continue
        }
        MsMsgReceived++
        fmt.Printf("new milestone attached: %s\n", msgSplit[1])
    }
}

func StartDoubleFeed(address string) {
    socket, err := zmq4.NewSocket(zmq4.SUB)
    must(err)
    socket.SetSubscribe("double")
    err = socket.Connect(address)
    must(err)

    fmt.Printf("started double-spending feed\n")
    for {
        msg, err := socket.Recv(0)
        must(err)

        msgSplit := strings.Split(msg, " ")
        if len(msgSplit) != 2 {
            fmt.Printf("double-spending: receive error! message format error\n")
            continue
        }
        
        fmt.Printf("double-spending detected. address: %s\n", msgSplit[1])
        // begin BFS
        effected_txs := 0
        que := list.New()
        for hash, tx := range transactions {
            if tx.Address == msgSplit[1] {
                que.PushBack(hash)
                d, has := doubles[hash]
                if has {
                    d.visited = true
                }
            }
        }
        for que.Len() != 0 {
            cur := que.Front()
            que.Remove(cur)
            fmt.Printf("current queue front: %s\n", cur.Value.(string))
            effected_txs++
            d, has := doubles[cur.Value.(string)]
            if has {
                for _, v := range d.TXs {
                    if !doubles[v.Hash].visited {
                        que.PushBack(v.Hash)
                        doubles[v.Hash].visited = true
                    }
                }
            }
        }
        fmt.Printf("double-spending effected %d txs to be reattached.\n", effected_txs)
    }
}

var Interval_txs []string
var Start_time int64 = time.Now().Unix()

func StartLog(interval int) {
    for {
        lastTotalTxs := TxMsgReceived
        Interval_txs = Interval_txs[0:0]

        time.Sleep(time.Duration(interval) * time.Second)

        var totalLatency      int64 = 0
        var totalInherent_lat int64 = 0
        var totalConfirm_lat  int64 = 0
        var total = 0
        for _, v := range Interval_txs {
            totalLatency += transactions[v].Inherent_lat + transactions[v].Confirm_lat
            totalInherent_lat += transactions[v].Inherent_lat
            totalConfirm_lat += transactions[v].Confirm_lat
            total++
        }

        a := time.Now().Unix() - Start_time - int64(interval)
        b := time.Now().Unix() - Start_time

        fmt.Printf("[%d s - %d s]: Average Latency %f,\n", a, b, float64(totalLatency)/float64(total))
        fmt.Printf("[%d s - %d s]: Including inherent latency %f and confirming latency %f.\n", a, b, float64(totalInherent_lat)/float64(total), float64(totalConfirm_lat)/float64(total))
        fmt.Printf("[%d s - %d s]: Average Throughput %f TPS.\n", a, b, float32(TxMsgReceived-lastTotalTxs)/float32(interval))
        fmt.Printf("[ 0 s - %d s]: Totally Tips ratio: %f, Confirmed ratio: %f.\n", b, float32(TotalTips)/float32(TxMsgReceived), float32(ConfirmedMsgReceived)/float32(TxMsgReceived))
        fmt.Printf("\n")
    }
}

func buildTxFromZMQData(msg string) *Transaction {
    msgSplit := strings.Split(msg, " ")
    if len(msgSplit) != 13 {
        if msgSplit[0] == "tx_trytes" {
            return &Transaction{Type:msgSplit[0]}
        } else {
            return nil
        }
    }
    var err error
    tx := &Transaction{}
    tx.Type = msgSplit[0]
    tx.Hash = msgSplit[1]
    tx.Address = msgSplit[2]
    tx.Value, err = strconv.Atoi(msgSplit[3])
    if err != nil {
        return nil
    }
    tx.ObsoleteTag = msgSplit[4]
    tx.Timestamp, err = strconv.ParseInt(msgSplit[5], 10, 64)
    if err != nil {
        return nil
    }
    tx.CurrentIndex, err = strconv.Atoi(msgSplit[6])
    if err != nil {
        return nil
    }
    tx.LastIndex, err = strconv.Atoi(msgSplit[7])
    if err != nil {
        return nil
    }
    tx.BundleHash = msgSplit[8]
    tx.TrunkTxHash = msgSplit[9]
    tx.BranchTxHash = msgSplit[10]
    tx.ArrivalTime, err = strconv.ParseInt(msgSplit[11], 10, 64)
    if err != nil {
        return nil
    }
    tx.Tag = msgSplit[12]
    return tx
}

func buildConfirmFromZMQData(msg string) *ConfTx {
    msgSplit := strings.Split(msg, " ")
    if len(msgSplit) != 7 {
        return nil
    }
    msgSplit = msgSplit[2:]
    tx := &ConfTx{}
    tx.Hash = msgSplit[0]
    tx.Address = msgSplit[1]
    tx.TrunkTxHash = msgSplit[2]
    tx.BranchTxHash = msgSplit[3]
    tx.BundleHash = msgSplit[4]
    return tx
}
