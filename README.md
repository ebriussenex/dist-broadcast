# Broadcasting distributed system

This repository presents a fast, fault-tolerant distributed broadcast system that allows you to broadcast messages to multiple nodes on any network topology. Configurable for different loads and different topologies.  A message that was sent to one node will quickly appear on all the others and temporary network problems will not lead to never-read or duplicated messages, due to asynchronous synchronization.  
The protocol is based on the maelstrom protocol and runs only with this protocol.  
  
For grid-topology on 25 nodes, with 100 messages per second and 100ms network latency, a result of 12 messages per operation is achieved with median latency < 1 second and worst latency < 2 seconds (~1600 ms) ([`syncDeadline`](https://github.com/ebriussenex/dist-broadcast/blob/master/main.go#L22) should be 30 ms to achieve this)  
  
There is a chance to achieve a better performance for different loads by configuration of [node](https://github.com/ebriussenex/dist-broadcast/blob/master/node/node.go) and [syncer](https://github.com/ebriussenex/dist-broadcast/blob/master/node/syncer.go), while keeping same code.  

For testing with maelstrom + jepsen:  
```bash

./maelstrom test -w broadcast --bin <your-binary-path> --node-count 25 --time-limit 20 --rate 100 --latency 100

```
