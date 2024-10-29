package node

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

type (
	// node callback function to make an rpc call with syncer data
	sendSyncFn       func(string, []int) error
	neighborSyncData struct {
		mu           *sync.Mutex
		lastTimeSync time.Time
		// this is needed only to undo marking if sync had failed
		prevTimeSync   time.Time
		messagesToSync map[int]struct{}
	}

	neighborSyncList struct {
		mu   *sync.RWMutex
		data map[string]*neighborSyncData
	}

	nodeSyncer struct {
		neighborsSyncList *neighborSyncList
		syncInterval      time.Duration
		syncDeadline      time.Duration
		batchSize         int
		neighbors         []string
		sendSync          sendSyncFn
	}
)

func NewNodeSyncer(syncInterval, syncDeadline time.Duration, batchSize int, sendSync sendSyncFn) *nodeSyncer {
	return &nodeSyncer{
		nil, syncInterval, syncDeadline, batchSize, nil, sendSync,
	}
}

func (n *nodeSyncer) initNeighbors(neighbors []string) {
	var once sync.Once
	once.Do(func() {
		syncList := make(map[string]*neighborSyncData, len(neighbors))
		for _, neighbor := range neighbors {
			syncList[neighbor] = &neighborSyncData{
				mu:             &sync.Mutex{},
				lastTimeSync:   time.Now(),
				prevTimeSync:   time.Now(),
				messagesToSync: map[int]struct{}{},
			}
		}

		n.neighborsSyncList = &neighborSyncList{
			mu:   &sync.RWMutex{},
			data: syncList,
		}

		n.neighbors = neighbors
	})
}

func (n *nodeSyncer) addMessages(neighbor string, msgs ...int) {
	n.neighborsSyncList.mu.RLock()
	neigborData, prs := n.neighborsSyncList.data[neighbor]
	n.neighborsSyncList.mu.RUnlock()

	if !prs {
		return
	}

	neigborData.mu.Lock()
	if neigborData.messagesToSync == nil {
		neigborData.messagesToSync = make(map[int]struct{}, len(msgs))
	}

	for _, msg := range msgs {
		neigborData.messagesToSync[msg] = struct{}{}
	}
	neigborData.mu.Unlock()
}

func (n *nodeSyncer) fetchAndClear(neighbor string) []int {
	n.neighborsSyncList.mu.RLock()
	neighborData, prs := n.neighborsSyncList.data[neighbor]
	n.neighborsSyncList.mu.RUnlock()
	if !prs {
		return make([]int, 0)
	}

	neighborData.mu.Lock()
	defer neighborData.mu.Unlock()

	messagesToSync := make([]int, 0, len(neighborData.messagesToSync))

	for msg := range neighborData.messagesToSync {
		messagesToSync = append(messagesToSync, msg)
	}

	neighborData.messagesToSync = make(map[int]struct{})
	neighborData.prevTimeSync = neighborData.lastTimeSync
	neighborData.lastTimeSync = time.Now()

	return messagesToSync
}

func (n *nodeSyncer) removeKnownMessages(neighbor string, msgs []int) {
	n.neighborsSyncList.mu.RLock()
	neighborData, prs := n.neighborsSyncList.data[neighbor]
	n.neighborsSyncList.mu.RUnlock()
	if !prs {
		return
	}

	neighborData.mu.Lock()
	defer neighborData.mu.Unlock()

	for _, msg := range msgs {
		delete(neighborData.messagesToSync, msg)
	}
}

// main loop of syncing
func (n *nodeSyncer) serveSync(ctx context.Context) {
	for n.neighborsSyncList == nil {
		<-time.After(n.syncInterval)
		slog.Info("trying to sync, but syncer has not been initialized")
	}

	if n.neighbors == nil {
		slog.Error("list of neighbors is not initialized, when sync list initialized")
		return
	}

	slog.Info("starting sync loop")

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(n.syncInterval):
			n.syncNeighbors()
		}
	}
}

func (n *nodeSyncer) syncNeighbors() {
	var wg sync.WaitGroup
	wg.Add(len(n.neighbors))
	errChan := make(chan error)

	for _, neighbor := range n.neighbors {
		go func(neighbor string) {
			defer wg.Done()

			n.neighborsSyncList.mu.RLock()
			neighborData, prs := n.neighborsSyncList.data[neighbor]
			n.neighborsSyncList.mu.RUnlock()
			if !prs || !n.readyToSync(neighborData) {
				return
			}

			msgsToSync := n.fetchAndClear(neighbor)
			if err := n.sendSync(neighbor, msgsToSync); err != nil {
				errChan <- err
				// if sync failed it is important to save messages been tried to synced, while keeping messages aqcuired in between
				n.addMessages(neighbor, msgsToSync...)
				neighborData.lastTimeSync = neighborData.prevTimeSync
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
		slog.Warn("failed to sync with one ore more neighbors", "err", errors.Join(errs...))
	}
}

func (n *nodeSyncer) readyToSync(neighborData *neighborSyncData) bool {
	neighborData.mu.Lock()
	defer neighborData.mu.Unlock()
	return time.Since(neighborData.lastTimeSync) > n.syncDeadline || len(neighborData.messagesToSync) >= n.batchSize
}
