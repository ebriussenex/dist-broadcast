package main

import (
	"log"
	"os"
	"time"

	"github.com/ebriussenex/dist-broadcast/node"
	retry "github.com/ebriussenex/dist-broadcast/pkg"
	"github.com/ebriussenex/dist-broadcast/storage"
	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	storage := storage.Init[int]()
	n := maelstrom.NewNode()

	retryFunc := func(fn func() error) error {
		return retry.FixedInterval(fn, 5, time.Second)
	}
	h := node.New(n, storage, time.Second*5, retryFunc)

	n.Handle("broadcast", h.HandleBroadcast)
	n.Handle("read", h.HandleRead)
	n.Handle("topology", h.HandleTopology)

	if err := n.Run(); err != nil {
		log.Printf("ERROR: %s", err)
		os.Exit(1)
	}

}
