package subpub

import (
	"context"
	"errors"
	"sync"
)

type MessageHandler func(msg interface{})

type Subscription interface {
	Unsubscribe()
}

type SubPub interface {
	Subscribe(subject string, cb MessageHandler) (Subscription, error)
	Publish(subject string, msg interface{}) error
	Close(ctx context.Context) error
}

type subscription struct {
	subject string
	handler MessageHandler
	sp      *subPubImpl
}

func (s *subscription) Unsubscribe() {
	s.sp.unsubscribe(s)
}

type subPubImpl struct {
	mu          sync.RWMutex
	subjects    map[string][]*subscription
	closed      bool
	publishChan chan publishRequest
	wg          sync.WaitGroup
}

type publishRequest struct {
	subject string
	msg     interface{}
}

func New() SubPub {
	sp := &subPubImpl{
		subjects:    make(map[string][]*subscription),
		publishChan: make(chan publishRequest, 100),
	}

	sp.wg.Add(1)
	go sp.publisher()

	return sp
}

func (sp *subPubImpl) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.closed {
		return nil, errors.New("subpub is closed")
	}

	sub := &subscription{
		subject: subject,
		handler: cb,
		sp:      sp,
	}

	sp.subjects[subject] = append(sp.subjects[subject], sub)
	return sub, nil
}

func (sp *subPubImpl) unsubscribe(sub *subscription) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	subs, ok := sp.subjects[sub.subject]
	if !ok {
		return
	}

	for i, s := range subs {
		if s == sub {
			sp.subjects[sub.subject] = append(subs[:i], subs[i+1:]...)
			break
		}
	}

	if len(sp.subjects[sub.subject]) == 0 {
		delete(sp.subjects, sub.subject)
	}
}

func (sp *subPubImpl) Publish(subject string, msg interface{}) error {
	sp.mu.RLock()
	defer sp.mu.RUnlock()

	if sp.closed {
		return errors.New("subpub is closed")
	}

	sp.publishChan <- publishRequest{subject: subject, msg: msg}
	return nil
}

func (sp *subPubImpl) publisher() {
	defer sp.wg.Done()

	for req := range sp.publishChan {
		sp.mu.RLock()
		subs, ok := sp.subjects[req.subject]
		if !ok {
			sp.mu.RUnlock()
			continue
		}

		subscribers := make([]*subscription, len(subs))
		copy(subscribers, subs)
		sp.mu.RUnlock()

		for _, sub := range subscribers {
			sub := sub
			go func() {
				sub.handler(req.msg)
			}()
		}
	}
}

func (sp *subPubImpl) Close(ctx context.Context) error {
	sp.mu.Lock()
	if sp.closed {
		sp.mu.Unlock()
		return nil
	}
	sp.closed = true
	close(sp.publishChan)
	sp.mu.Unlock()

	done := make(chan struct{})
	go func() {
		sp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
