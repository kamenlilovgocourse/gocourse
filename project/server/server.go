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
}

var (
	nextClientId int64
	mapsLock     [item.IDMapsCount]sync.Mutex
	maps         [item.IDMapsCount]map[string]mapEntry
)

func init() {
	for i := 0; i < item.IDMapsCount; i++ {
		maps[i] = make(map[string]mapEntry)
	}
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
	if p.Expiry != nil {
		exp := p.Expiry.AsTime()
		me.Expiry = &exp
	}
	mapsLock[hash].Lock()
	maps[hash][as.Compose()] = me
	mapsLock[hash].Unlock()
	if p.Expiry != nil {
		insertInExpList(&as, p.Expiry.AsTime())
	}
	ret := &cachegrpc.SetItemResult{}
	return ret, nil
}

func (s *CacheServer) GetItem(ctx context.Context, p *cachegrpc.GetItemParams) (*cachegrpc.GetItemResult, error) {
	as := item.ID{Owner: p.Owner, Service: p.Service, Name: p.Name}
	fmt.Printf("Owner: %s Service: %s, Name: %s\n", p.Owner, p.Service, p.Name)
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
