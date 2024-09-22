package main

import (
	"log"
	"os"
	"time"

	"github.com/ebriussenex/dist-broadcast/node"
	"github.com/ebriussenex/dist-broadcast/storage"
	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	storage := storage.Init[int]()
	n := maelstrom.NewNode()
	h := node.New(n, storage, time.Second*5)

	n.Handle("broadcast", h.HandleBroadcast)
	n.Handle("read", h.HandleRead)
	n.Handle("topology", h.HandleTopology)

	if err := n.Run(); err != nil {
		log.Printf("ERROR: %s", err)
		os.Exit(1)
	}

}
