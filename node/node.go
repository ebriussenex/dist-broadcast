package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ebriussenex/dist-broadcast/message"
	"github.com/ebriussenex/dist-broadcast/storage"
	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type (
	Storage interface {
		Add(int)
		GetAll() []int
		Present(int) bool
	}

	Node struct {
		storage   Storage
		neighbors []string
		node      *maelstrom.Node

		waitForResponse time.Duration
	}
)

func New(node *maelstrom.Node, storage *storage.ConcurrentSet[int], waitForResponse time.Duration) Node {
	return Node{
		storage, make([]string, 0), node, waitForResponse,
	}
}

func (n *Node) broadcastMsg(msg maelstrom.Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), n.waitForResponse)
	defer cancel()

	errChan := make(chan error)

	var wg sync.WaitGroup
	wg.Add(len(n.neighbors))

	for _, neighbor := range n.neighbors {
		go func(neighbor string) {
			defer wg.Done()
			response, err := n.node.SyncRPC(ctx, neighbor, msg.Body)
			if err != nil {
				errChan <- fmt.Errorf("rpc to neighbor: %s failed: %w", neighbor, err)
				return
			}

			if response.Type() != "broadcast_ok" {
				errChan <- fmt.Errorf("unexpected message status: neighbor: %s, type: %s", neighbor, response.Type())
				return
			}

		}(neighbor)
	}
	go func() {
		wg.Wait()
		close(errChan)
	}()

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) != 0 {
		return fmt.Errorf("one ore more broadcasts failed: %w", errors.Join(errs...))
	}

	return nil
}

func (n *Node) HandleBroadcast(msg maelstrom.Message) error {
	var request message.BroadcastReq
	if err := json.Unmarshal(msg.Body, &request); err != nil {
		return fmt.Errorf("failed to unmarshall request: %w", err)
	}

	if n.storage.Present(request.Message) {
		return nil
	}

	n.storage.Add(request.Message)

	if err := n.node.Reply(msg, message.BroadcastResp{
		Type:      "broadcast_ok",
		InReplyTo: request.MsgID,
	}); err != nil {
		return fmt.Errorf("failed to reply to broadcast msg: %w", err)
	}

	if err := n.broadcastMsg(msg); err != nil {
		return fmt.Errorf("broadcasting to neighbors failed: %w", err)
	}

	return nil
}

func (n *Node) HandleRead(msg maelstrom.Message) error {
	var request message.ReadReq
	if err := json.Unmarshal(msg.Body, &request); err != nil {
		return fmt.Errorf("failed to unmarshall request: %w", err)
	}

	messages := n.storage.GetAll()

	return n.node.Reply(msg, message.ReadResp{
		Type:     "read_ok",
		Messages: messages,
	})
}

func (n *Node) HandleTopology(msg maelstrom.Message) error {
	var request message.TopologyReq
	if err := json.Unmarshal(msg.Body, &request); err != nil {
		return fmt.Errorf("failed to unmarshall request: %w", err)
	}

	selfNodeIS := n.node.ID()
	if neighbors, ok := request.Topology[selfNodeIS]; ok {
		n.neighbors = neighbors
	}

	return n.node.Reply(msg, message.TopologyResp{
		Type: "topology_ok",
	})
}
