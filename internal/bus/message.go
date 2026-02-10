package bus

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	ChannelTelegram  = "telegram"
	ChannelWebSocket = "websocket"
	ChannelCLI       = "cli"
)

type Message struct {
	ID        string
	Channel   string
	ChatID    string
	Content   string
	Timestamp time.Time
	Metadata  map[string]interface{}
}

type MessageHandler func(ctx context.Context, msg *Message) error

type MessageBus interface {
	Publish(ctx context.Context, channel string, msg *Message) error
	Subscribe(channel string, handler MessageHandler) (string, error)
	Unsubscribe(channel string, handlerID string) error
	Close() error
}

type InMemoryMessageBus struct {
	mu          sync.RWMutex
	subscribers map[string]map[string]MessageHandler
	messageCh   chan *Message
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func NewInMemoryMessageBus(ctx context.Context) *InMemoryMessageBus {
	busCtx, cancel := context.WithCancel(ctx)
	return &InMemoryMessageBus{
		subscribers: make(map[string]map[string]MessageHandler),
		messageCh:   make(chan *Message, 100),
		ctx:         busCtx,
		cancel:      cancel,
	}
}

func (b *InMemoryMessageBus) Start() {
	b.wg.Add(1)
	go b.processMessages()
}

func (b *InMemoryMessageBus) processMessages() {
	defer b.wg.Done()

	for {
		select {
		case <-b.ctx.Done():
			return
		case msg := <-b.messageCh:
			b.mu.RLock()
			handlers, ok := b.subscribers[msg.Channel]
			b.mu.RUnlock()

			if ok {
				for _, handler := range handlers {
					b.wg.Add(1)
					go func(h MessageHandler) {
						defer b.wg.Done()
						if err := h(b.ctx, msg); err != nil {
							fmt.Printf("Handler error: %v\n", err)
						}
					}(handler)
				}
			}
		}
	}
}

func (b *InMemoryMessageBus) Publish(ctx context.Context, channel string, msg *Message) error {
	msg.Channel = channel
	msg.Timestamp = time.Now()

	select {
	case b.messageCh <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return ErrTimeout
	}
}

func (b *InMemoryMessageBus) Subscribe(channel string, handler MessageHandler) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.subscribers[channel]; !ok {
		b.subscribers[channel] = make(map[string]MessageHandler)
	}

	handlerID := fmt.Sprintf("%s-%d-%d", channel, time.Now().UnixNano(), len(b.subscribers[channel]))
	b.subscribers[channel][handlerID] = handler

	return handlerID, nil
}

func (b *InMemoryMessageBus) Unsubscribe(channel string, handlerID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if handlers, ok := b.subscribers[channel]; ok {
		if _, exists := handlers[handlerID]; exists {
			delete(handlers, handlerID)
			return nil
		}
	}
	return ErrHandlerNotFound
}

func (b *InMemoryMessageBus) Close() error {
	b.cancel()
	b.wg.Wait()
	close(b.messageCh)
	return nil
}
