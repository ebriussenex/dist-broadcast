package node

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/ebriussenex/dist-broadcast/message"
	"github.com/ebriussenex/dist-broadcast/storage"
	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type (
	Storage interface {
		Add(...int)
		GetAll() []int
		Present(int) bool
		Size() int
		Clear()
		Delete(int)
	}

	Node struct {
		ctx       context.Context
		storage   Storage
		neighbors []string
		node      *maelstrom.Node

		waitForResponse time.Duration
		withRetry       retryFunc
		nodeSyncer      *nodeSyncer
	}

	retryFunc func(func() error) error
)

func New(
	ctx context.Context,
	maelstromNode *maelstrom.Node,
	waitForResponse time.Duration,
	withRetry retryFunc,
	syncInterval time.Duration,
	syncDeadline time.Duration,
	batchSize int,
) *Node {
	node := &Node{
		ctx:             ctx,
		storage:         storage.Init[int](),
		neighbors:       make([]string, 0),
		node:            maelstromNode,
		waitForResponse: waitForResponse,
		withRetry:       withRetry,
	}

	sendSync := func(neighbor string, msgs []int) error {
		ctx, cancel := context.WithTimeout(node.ctx, waitForResponse)
		defer cancel()
		return withRetry(func() error {
			err := node.syncWithNeighbor(ctx, neighbor, msgs)
			if err != nil {
				slog.Default().With("neighbor", neighbor).Error("failed sync with neighbor", "err", err)
			}
			return err
		})
	}

	node.nodeSyncer = NewNodeSyncer(syncInterval, syncDeadline, batchSize, sendSync)
	go node.nodeSyncer.serveSync(ctx)

	return node
}

func (n *Node) syncWithNeighbor(ctx context.Context, neighbor string, messages []int) error {
	syncMessage := message.SyncReq{
		Type:     "sync",
		Messages: messages,
	}

	response, err := n.node.SyncRPC(ctx, neighbor, syncMessage)
	if err != nil {
		return fmt.Errorf("rpc to neighbor: %s failed: %w", neighbor, err)
	}

	if response.Type() != "sync_ok" {
		return fmt.Errorf("unexpected message status: neighbor: %s, type: %s", neighbor, response.Type())
	}
	return nil
}

func (n *Node) HandleSync(syncMsg maelstrom.Message) error {
	err := n.withRetry(func() error {
		return n.node.Reply(syncMsg, message.SyncResp{
			Type: "sync_ok",
		})
	})

	if err != nil {
		slog.Error("failed to reply to sync", "err", err)
		return err
	}

	var syncReq message.SyncReq
	if err := json.Unmarshal(syncMsg.Body, &syncReq); err != nil {
		return fmt.Errorf("failed to unmarshall request: %w", err)
	}

	fromNeighbor := syncMsg.Src
	n.nodeSyncer.removeKnownMessages(fromNeighbor, syncReq.Messages)

	n.storage.Add(syncReq.Messages...)

	for _, neighbor := range n.neighbors {
		if neighbor == syncMsg.Src {
			continue
		}
		n.nodeSyncer.addMessages(neighbor, syncReq.Messages...)
	}

	return err
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
	for _, neighbor := range n.neighbors {
		n.nodeSyncer.addMessages(neighbor, request.Message)
	}

	err := n.withRetry(func() error {
		return n.acknowledgeBroadcast(msg)
	})

	if err != nil {
		slog.Error("failed to acknowledge broadcast", "err", err)
		return fmt.Errorf("failed to acknowledge broadcast: %w", err)
	}

	return nil
}

func (n *Node) acknowledgeBroadcast(msg maelstrom.Message) error {
	if err := n.node.Reply(msg, message.BroadcastResp{
		Type: "broadcast_ok",
	}); err != nil {
		return err
	}
	return nil
}

func (n *Node) HandleRead(msg maelstrom.Message) error {
	var request message.ReadReq
	if err := json.Unmarshal(msg.Body, &request); err != nil {
		return fmt.Errorf("failed to unmarshall request: %w", err)
	}

	messages := n.storage.GetAll()

	err := n.withRetry(func() error {
		return n.node.Reply(msg, message.ReadResp{
			Type:     "read_ok",
			Messages: messages,
		})
	})

	if err != nil {
		slog.Error("failed to reply read", "err", err)
	}

	return err
}

func (n *Node) HandleTopology(msg maelstrom.Message) error {
	var request message.TopologyReq
	if err := json.Unmarshal(msg.Body, &request); err != nil {
		return fmt.Errorf("failed to unmarshall request: %w", err)
	}

	selfNodeID := n.node.ID()
	if neighbors, ok := request.Topology[selfNodeID]; ok {
		n.neighbors = neighbors
		n.nodeSyncer.initNeighbors(neighbors)
	}

	err := n.withRetry(func() error {
		return n.node.Reply(msg, message.TopologyResp{
			Type: "topology_ok",
		})
	})

	if err != nil {
		slog.Error("failed to reply topology", "err", err)
	}

	return err
}
