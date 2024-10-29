package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/ebriussenex/dist-broadcast/node"
	retry "github.com/ebriussenex/dist-broadcast/pkg"
	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()
	ctx := context.Background()

	retryFunc := func(fn func() error) error {
		return retry.FixedInterval(fn, 5, time.Millisecond*100)
	}
	syncInterval := time.Millisecond * 50
	syncDeadline := time.Millisecond * 300
	h := node.New(ctx, n, time.Second*3, retryFunc, syncInterval, syncDeadline, 5)

	n.Handle("broadcast", h.HandleBroadcast)
	n.Handle("read", h.HandleRead)
	n.Handle("topology", h.HandleTopology)
	n.Handle("sync", h.HandleSync)

	if err := n.Run(); err != nil {
		log.Printf("ERROR: %s", err)
		os.Exit(1)
	}

}
