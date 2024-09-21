package node

import (
	"encoding/json"
	"fmt"

	"github.com/ebriussenex/dist-broadcast/message"
	"github.com/ebriussenex/dist-broadcast/storage"
	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type Storage interface {
	Add(int)
	GetAll() []int
}

type Node struct {
	storage   Storage
	neighbors []string
	node      *maelstrom.Node
}

func New(node *maelstrom.Node, storage *storage.ConcurrentSet[int]) Node {
	return Node{
		storage, make([]string, 0), node,
	}
}

func (n *Node) broadcastMsg(msg maelstrom.Message) {
	for _, neighbor := range n.neighbors {
		err := n.node.RPC(neighbor, msg, func(response maelstrom.Message) {

		})
		if err != nil {
			return fmt.Errorf("failed to init RPC to node: %s, err: %w", err)
		}
	}
	n.node.RPC()
}

func (n *Node) BroadcastWaiter() {

}

func (n *Node) HandleBroadcast(msg maelstrom.Message) error {
	var request message.BroadcastReq
	if err := json.Unmarshal(msg.Body, &request); err != nil {
		return fmt.Errorf("failed to unmarshall request: %w", err)
	}

	n.storage.Add(request.Message)

	if err := n.node.Reply(msg, message.BroadcastResp{
		Type: "broadcast_ok",
	}); err != nil {
		return fmt.Errorf("failed to handle broadcast %w", err)
	}

	n.broadcastMsg(msg)

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
