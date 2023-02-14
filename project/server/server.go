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

type CacheServer struct {
	cachegrpc.UnimplementedCacheServerServer
}

func NewServer() *CacheServer {
	s := &CacheServer{}
	return s
}

func (s *CacheServer) GetClientID(context.Context, *cachegrpc.AssignClientID) (*cachegrpc.AssignedClientID, error) {
	ret := &cachegrpc.AssignedClientID{}
	ret.Id = fmt.Sprintf("%d", atomic.AddInt64(&nextClientId, 1))
	return ret, nil
}

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

func remove(slice []chan struct{}, s chan struct{}) []chan struct{} {
	for i := 0; i < len(slice); i++ {
		if slice[i] == s {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return nil
}

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
