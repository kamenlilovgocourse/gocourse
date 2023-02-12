package server

import (
	"log"
	"sync"
	"time"

	"github.com/kamenlilovgocourse/gocourse/project/item"
)

type expListEntry struct {
	next   *expListEntry
	ID     item.ID
	Expiry time.Time
}

var (
	expListLock                sync.Mutex
	expList                    *expListEntry
	InsertThreadShutdown       sync.WaitGroup
	NotifyInsertThreadShutdown chan struct{}
)

func insertInExpList(Id *item.ID, exp time.Time) {
	expListLock.Lock()
	defer expListLock.Unlock()
	var prevl *expListEntry = nil
	l := expList
	for l != nil {
		if exp.After(l.Expiry) {
			prevl = l
			l = l.next
			continue
		}
		// This time is before the list item, insert it here
		newEntry := expListEntry{ID: *Id, Expiry: exp, next: l}
		if prevl == nil {
			expList = &newEntry
		} else {
			prevl.next = &newEntry
		}
		return
	}
	// This time should go at the end of the list
	newEntry := expListEntry{ID: *Id, Expiry: exp, next: nil}
	if prevl == nil {
		expList = &newEntry
	} else {
		prevl.next = &newEntry
	}
}

func ScanExpListRoutine() {
	for {
		time.Sleep(time.Second)
		now := time.Now().UTC()
		expListLock.Lock()
		for expList != nil {
			log.Printf("comparing now=%v item=%v\n", now, expList.Expiry)
			if expList.Expiry.After(now) {
				// No need to scan further, there's yet time for this item
				break
			}
			// Remove this item
			log.Printf("Removing stale item %s at %v\n", expList.ID.Compose(), now)
			expList = expList.next
			// Rescan, maybe more items are expired
		}
		expListLock.Unlock()
		select {
		case <-NotifyInsertThreadShutdown:
			InsertThreadShutdown.Done()
			return
		default:
			continue
		}
	}
}
