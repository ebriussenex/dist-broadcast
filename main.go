package main

import (
	"log"
	"os"

	"github.com/ebriussenex/dist-broadcast/handler"
	"github.com/ebriussenex/dist-broadcast/storage"
	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	storage := storage.Init[int]()
	n := maelstrom.NewNode()
	h := handler.New(n, storage)

	n.Handle("broadcast", h.HandleBroadcast)
	n.Handle("read", h.HandleRead)
	n.Handle("topology", h.HandleTopology)

	if err := n.Run(); err != nil {
		log.Printf("ERROR: %s", err)
		os.Exit(1)
	}

}
