package server

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kamenlilovgocourse/gocourse/project/cachegrpc"
	"github.com/kamenlilovgocourse/gocourse/project/item"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// A map entry consists of the value of the cache item, an optional expiry,
// plus a (empty or nonempty) slice of subscriptions. If subscriptions are
// present, every time the value is updated, the appropriate subscriber
// listeners goroutines are notified via the channel so they can generate
// a push notification to a connected client
type mapEntry struct {
	Value  string
	Expiry *time.Time
	Subs   []chan struct{}
}

var (
	nextClientId   int64
	mapsLock       [item.IDMapsCount]sync.Mutex
	maps           [item.IDMapsCount]map[string]mapEntry
	StopServerChan chan struct{}
)

func init() {
	for i := 0; i < item.IDMapsCount; i++ {
		maps[i] = make(map[string]mapEntry)
	}
	StopServerChan = make(chan struct{})
}

// The gRPC CacheServer service with out own implementations
type CacheServer struct {
	cachegrpc.UnimplementedCacheServerServer
}

func NewServer() *CacheServer {
	s := &CacheServer{}
	return s
}

// GetClientID uses a globally unique, incrementing int counter to provide unique IDs to cache
// clients that are interested in having one
func (s *CacheServer) GetClientID(context.Context, *cachegrpc.AssignClientID) (*cachegrpc.AssignedClientID, error) {
	ret := &cachegrpc.AssignedClientID{}
	ret.Id = fmt.Sprintf("%d", atomic.AddInt64(&nextClientId, 1))
	return ret, nil
}

// SetItem sets a cache item with a given ID, value and optional expiry on the server. If the
// item value was already set, and any subscribers are attached to it, they are notified that the
// value is updated so they can push a notification to a connected client
func (s *CacheServer) SetItem(ctx context.Context, p *cachegrpc.SetItemParams) (*cachegrpc.SetItemResult, error) {
	as := item.ID{Owner: p.Owner, Service: p.Service, Name: p.Name}
	hash := as.HashKey()
	me := mapEntry{Value: p.Value}
	me.Subs = make([]chan struct{}, 0)
	if p.Expiry != nil {
		exp := p.Expiry.AsTime()
		me.Expiry = &exp
	}
	mapsLock[hash].Lock()
	prevMe, found := maps[hash][as.Compose()]
	if found {
		me.Subs = prevMe.Subs
	}
	maps[hash][as.Compose()] = me
	mapsLock[hash].Unlock()
	if p.Expiry != nil {
		insertInExpList(&as, p.Expiry.AsTime())
	}
	for _, notify := range me.Subs {
		notify <- struct{}{}
	}
	ret := &cachegrpc.SetItemResult{}
	return ret, nil
}

// Retrieve the value of a previously set cache item
func (s *CacheServer) GetItem(ctx context.Context, p *cachegrpc.GetItemParams) (*cachegrpc.GetItemResult, error) {
	as := item.ID{Owner: p.Owner, Service: p.Service, Name: p.Name}
	hash := as.HashKey()
	mapsLock[hash].Lock()
	result, ok := maps[hash][as.Compose()]
	mapsLock[hash].Unlock()
	resultFmt := cachegrpc.GetItemResult{}
	if !ok {
		return &resultFmt, errors.New("Item " + as.Compose() + " not found")
	}
	resultFmt.Value = result.Value
	if result.Expiry != nil {
		resultFmt.Expiry = timestamppb.New(*result.Expiry)
	}
	return &resultFmt, nil
}

// Helper routine: given a chan struct{}, and a slice of chan structs, locate
// the entry and remove it from the slice, returning the new slice
func remove(slice []chan struct{}, s chan struct{}) []chan struct{} {
	for i := 0; i < len(slice); i++ {
		if slice[i] == s {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return nil
}

// Service the SubscribeItem API call. This will typically be invoked by a client from a dedicated
// goroutine that will expect the server to occasionally send it notifications that the item with
// the specified ID has been updated, and this routine will send the updated value
func (s *CacheServer) SubscribeItem(p *cachegrpc.GetItemParams, stream cachegrpc.CacheServer_SubscribeItemServer) error {
	as := item.ID{Owner: p.Owner, Service: p.Service, Name: p.Name}
	hash := as.HashKey()
	var thisChan = make(chan struct{})
	mapsLock[hash].Lock()
	e, found := maps[hash][as.Compose()]
	if !found {
		e = mapEntry{Value: "", Expiry: nil}
	}
	e.Subs = append(maps[hash][as.Compose()].Subs, thisChan)
	maps[hash][as.Compose()] = e
	mapsLock[hash].Unlock()
	for {
		select {
		case <-thisChan:
			// Go on
		case <-StopServerChan:
			return nil
		}
		mapsLock[hash].Lock()
		e := maps[hash][as.Compose()]
		item := cachegrpc.GetItemResult{Value: e.Value}
		if e.Expiry != nil {
			item.Expiry = timestamppb.New(*e.Expiry)
		}
		mapsLock[hash].Unlock()
		err := stream.Send(&item)
		if err != nil {
			mapsLock[hash].Lock()
			e := maps[hash][as.Compose()]
			e.Subs = remove(e.Subs, thisChan)
			mapsLock[hash].Unlock()
			return err
		}
	}
}
