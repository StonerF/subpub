// package subpub

// import (
// 	"context"
// 	"sync"

// 	"github.com/google/uuid"
// )

// type Hub struct {
// 	mu   sync.Mutex
// 	subs map[string][]*Subscriber
// }

// func NewHub() *Hub {
// 	return &Hub{
// 		subs: make(map[string][]*Subscriber),
// 	}
// }

// type Subscriber struct {
// 	id       string
// 	mu       sync.Mutex
// 	messages chan *MessageHandler
// 	Handler  MessageHandler
// 	quit     chan struct{}
// }

// func (h *Hub) Subscribe(ctx context.Context, subject string, cb MessageHandler) (Subscription, error) {

// 	h.mu.Lock()
// 	h.subs[subject] = append(h.subs[subject], &Subscriber{id: uuid.NewString(), messages: make(chan *MessageHandler, 100), Handler: cb, quit: make(chan struct{})})
// 	h.mu.Unlock()

// 	ctx, cancel := context.WithCancel(ctx)
// 	defer cancel()
// 	go h.subs[subject][len(h.subs[subject])-1].run(ctx)
// 	return h.subs[subject][len(h.subs[subject])-1], nil

// }

// func (s *Subscriber) Unsubscribe() {
// 	close(s.quit)
// }

// func (s *Subscriber) run(ctx context.Context) {
// 	for {
// 		select {
// 		case msg := <-s.messages:
// 			s.Handler(msg)
// 		case <-s.quit:
// 			return
// 		case <-ctx.Done():
// 			return
// 		}
// 	}
// }

// type eventBus struct {
// 	mu          sync.RWMutex
// 	subscribers map[string][]*messageHandler
// 	closed      bool
// }

// func NewSubPub() SubPub {
// 	return &eventBus{
// 		subscribers: make(map[string][]*messageHandler),
// 	}
// }

// func (eb *eventBus) Subscribe(subject string, cb MessageHandler) (Subscription, error) {
// 	eb.mu.Lock()
// 	defer eb.mu.Unlock()

// 	if eb.closed {
// 		return nil, context.Canceled
// 	}

// 	ctx, cancel := context.WithCancel(context.Background())
// 	handler := &messageHandler{
// 		cb:  cb,
// 		ctx: ctx,
// 	}

// 	eb.subscribers[subject] = append(eb.subscribers[subject], handler)

// 	return &subscription{
// 		cancel:  cancel,
// 		subject: subject,
// 		handler: handler,
// 		bus:     eb,
// 	}, nil
// }

// func (eb *eventBus) Subscribe(ctx context.Context, subject string, cb MessageHandler) (Subscription, error) {
// 	eb.mu.Lock()
// 	defer eb.mu.Unlock()

// 	if eb.closed {
// 		return nil, errors.New("bus is closed")
// 	}

// 	// Создаем дочерний контекст с возможностью отмены
// 	subCtx, _ := context.WithCancel(ctx)

// 	handler := &messageHandler{
// 		cb:  cb,
// 		ctx: subCtx,
// 	}

// 	// Добавляем подписчика
// 	eb.subscribers[subject] = append(eb.subscribers[subject], handler)

// 	// Автоматическая отписка при завершении контекста
// 	go func() {
// 		<-subCtx.Done()
// 		eb.unsubscribe(subject, handler)
// 	}()

// 	return &Subscription{
// 		handler: handler,
// 		subject: subject,
// 		bus:     eb,
// 	}, nil
// }

// func (eb *eventBus) Close(ctx context.Context) error {
// 	eb.mu.Lock()
// 	eb.closed = true
// 	initialSubs := len(eb.subscribers)
// 	eb.mu.Unlock()

// 	if initialSubs == 0 {
// 		return nil
// 	}

// 	// Ожидаем завершения обработки сообщений
// 	ticker := time.NewTicker(100 * time.Millisecond)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return ctx.Err()
// 		case <-ticker.C:
// 			eb.mu.RLock()
// 			remaining := len(eb.subscribers)
// 			eb.mu.RUnlock()
// 			if remaining == 0 {
// 				return nil
// 			}
// 		}
// 	}
// }

// type subscription struct {
// 	cancel  context.CancelFunc
// 	subject string
// 	handler *messageHandler
// 	bus     *eventBus
// }

// func (eb *eventBus) Publish(ctx context.Context, subject string, msg interface{}) error {
// 	eb.mu.RLock()
// 	defer eb.mu.RUnlock()

// 	if eb.closed {
// 		return errors.New("bus is closed")
// 	}

// 	handlers := eb.subscribers[subject]
// 	if len(handlers) == 0 {
// 		return nil
// 	}

// 	// Создаем копию списка для безопасной обработки
// 	handlersCopy := make([]*messageHandler, len(handlers))
// 	copy(handlersCopy, handlers)

// 	for _, h := range handlersCopy {
// 		go func(h *messageHandler) {
// 			select {
// 			case <-h.ctx.Done(): // Проверяем актуальность подписки
// 			default:
// 				// Защита от паники в обработчике
// 				defer func() {
// 					if r := recover(); r != nil {
// 						log.Printf("handler panic: %v", r)
// 					}
// 				}()
// 				h.cb(msg)
// 			}
// 		}(h)
// 	}

// 	return nil
// }

// // Unsubscribe с полной блокировкой
// func (eb *eventBus) unsubscribe(subject string, handler *messageHandler) {
// 	eb.mu.Lock()
// 	defer eb.mu.Unlock()

// 	handlers := eb.subscribers[subject]
// 	for i, h := range handlers {
// 		if h == handler {
// 			// Удаляем и предотвращаем утечку памяти
// 			handlers[i] = nil
// 			eb.subscribers[subject] = append(handlers[:i], handlers[i+1:]...)
// 			break
// 		}
// 	}
// }

// Хранить в мапе отображения вида <топик> <-> []*HandlerMessage

package subpub

import (
	"context"
	"slices"
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

type messageHandler struct {
	cb  MessageHandler
	ctx context.Context
}

type message struct {
	subject string
	data    interface{}
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
	msgChan     chan message
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func NewSubPub() SubPub {
	ctx, cancel := context.WithCancel(context.Background())
	sp := &subPub{
		subscribers: make(map[string][]*subscription),
		msgChan:     make(chan message, 100),
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

func (s *subPub) Publish(subject string, msg interface{}) error {
	select {
	case s.msgChan <- message{subject: subject, data: msg}:
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
			subs := s.subscribers[msg.subject]
			s.mu.RUnlock()

			for _, sub := range subs {
				sub.handler(msg.data)
			}

		case <-s.ctx.Done():
			return
		}
	}
}
