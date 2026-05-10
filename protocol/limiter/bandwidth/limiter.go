package bandwidth

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/sagernet/sing-box/adapter"
)

type Limiter interface {
	WaitN(ctx context.Context, n int) (err error)
}

type FlowKeysLimiter struct {
	limiter      Limiter
	connIDGetter ConnIDGetter

	waits map[string][]*wait
	conns map[string]int
	queue chan struct{}
	reset time.Time

	mtx sync.Mutex
}

func NewFlowKeysLimiter(connIDGetter ConnIDGetter, limiter Limiter) *FlowKeysLimiter {
	return &FlowKeysLimiter{
		limiter:      limiter,
		connIDGetter: connIDGetter,
		waits:        make(map[string][]*wait),
		conns:        make(map[string]int),
		queue:        make(chan struct{}, 1),
		reset:        time.Now().Add(time.Second),
	}
}

func (l *FlowKeysLimiter) WaitN(ctx context.Context, n int) error {
	id, _ := l.connIDGetter(ctx, adapter.ContextFrom(ctx))
	mainWait := &wait{ctx, make(chan struct{}), n}
	l.mtx.Lock()
	if waits, ok := l.waits[id]; ok {
		l.waits[id] = append(waits, mainWait)
	} else {
		l.waits[id] = []*wait{mainWait}
	}
	l.mtx.Unlock()
	select {
	case l.queue <- struct{}{}:
	case <-mainWait.finish:
		return nil
	case <-ctx.Done():
		l.mtx.Lock()
		for i, wait := range l.waits[id] {
			if wait == mainWait {
				l.waits[id] = slices.Delete(l.waits[id], i, i+1)
				close(wait.finish)
				break
			}
		}
		l.mtx.Unlock()
		return ctx.Err()
	}
	for {
		if ctx.Err() != nil {
			l.mtx.Lock()
			for i, wait := range l.waits[id] {
				if wait == mainWait {
					l.waits[id] = slices.Delete(l.waits[id], i, i+1)
					close(wait.finish)
					break
				}
			}
			l.mtx.Unlock()
			<-l.queue
			return ctx.Err()
		}
		now := time.Now()
		if l.reset.Compare(now) == -1 {
			clear(l.conns)
			l.reset = now.Add(time.Second)
		}
		l.mtx.Lock()
		var minConnId string
		var minN int
		for connID, waits := range l.waits {
			if len(waits) == 0 {
				continue
			}
			if n, ok := l.conns[connID]; ok {
				if minConnId == "" {
					minConnId = connID
					minN = n
					continue
				}
				if n+waits[0].n < minN {
					minConnId = connID
					minN = n
				}
			} else {
				l.conns[connID] = 0
				minConnId = connID
				break
			}
		}
		minWait := l.waits[minConnId][0]
		l.waits[minConnId][0] = nil
		l.waits[minConnId] = l.waits[minConnId][1:]
		if len(l.waits) == 0 {
			delete(l.waits, minConnId)
		}
		l.mtx.Unlock()
		err := l.limiter.WaitN(ctx, minWait.n)
		if err != nil {
			continue
		}
		l.conns[minConnId] = l.conns[minConnId] + minWait.n
		close(minWait.finish)
		if minWait == mainWait {
			<-l.queue
			return nil
		}
	}
}

type wait struct {
	ctx    context.Context
	finish chan struct{}
	n      int
}
