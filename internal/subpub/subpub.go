package subpub

import (
	"context"
	"slices"
	"sync"

	"github.com/StonerF/subpub/internal/model"
)

type MessageHandler func(msg interface{})

type Subscription interface {
	Unsubscribe()
	GetChannel() chan *model.Message
}

type SubPub interface {
	Subscribe(subject string, cb MessageHandler) (Subscription, error)
	Publish(subject string, msg interface{}) error
	Close(ctx context.Context) error
}

type messageHandler struct {
	cb  MessageHandler
	ctx context.Context
}

type subscription struct {
	subPub  *subPub
	subject string
	handler MessageHandler
}

func (s *subscription) Unsubscribe() {
	s.subPub.unsubscribe(s)
}

type subPub struct {
	mu          sync.RWMutex
	subscribers map[string][]*subscription
	msgChan     chan *model.Message
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func NewSubPub() SubPub {
	ctx, cancel := context.WithCancel(context.Background())
	sp := &subPub{
		subscribers: make(map[string][]*subscription),
		msgChan:     make(chan *model.Message, 100),
		ctx:         ctx,
		cancel:      cancel,
	}

	sp.wg.Add(1)
	go sp.dispatchMessages()
	return sp
}

func (s *subPub) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub := &subscription{
		subPub:  s,
		subject: subject,
		handler: cb,
	}

	s.subscribers[subject] = append(s.subscribers[subject], sub)
	return sub, nil
}

func (s *subscription) GetChannel() chan *model.Message {
	return s.subPub.msgChan
}

func (s *subPub) Publish(subject string, msg interface{}) error {
	select {
	case s.msgChan <- &model.Message{Topic: subject, Data: msg}:
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
	return nil
}

func (s *subPub) Close(ctx context.Context) error {
	s.cancel()
	done := make(chan struct{})

	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *subPub) unsubscribe(sub *subscription) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subs := s.subscribers[sub.subject]
	for i, v := range subs {
		if v == sub {
			s.subscribers[sub.subject] = slices.Delete(s.subscribers[sub.subject], i, i+1)
			break
		}
	}
}

func (s *subPub) dispatchMessages() {
	defer s.wg.Done()

	for {
		select {
		case msg := <-s.msgChan:
			s.mu.RLock()
			subs := s.subscribers[msg.Topic]
			s.mu.RUnlock()

			for _, sub := range subs {
				sub.handler(msg.Data)
			}

		case <-s.ctx.Done():
			return
		}
	}
}
