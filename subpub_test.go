package subpub

import (
	"context"
	"sync"
	"testing"
	"time"

	"go.uber.org/goleak"
)

func TestSubscribePublish(t *testing.T) {
	defer goleak.VerifyNone(t)
	sp := NewSubPub()
	defer sp.Close(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)

	handler := func(msg interface{}) {
		defer wg.Done()
		if msg != "test message" {
			t.Errorf("Expected 'test message', got %v", msg)
		}
	}

	_, err := sp.Subscribe("test.subject", handler)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	err = sp.Publish("test.subject", "test message")
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	wg.Wait() // Wait for message to be processed
}

func TestUnsubscribe(t *testing.T) {
	defer goleak.VerifyNone(t)
	sp := NewSubPub()
	defer sp.Close(context.Background())

	called := false
	handler := func(msg interface{}) {
		called = true
	}

	sub, _ := sp.Subscribe("test.subject", handler)
	sub.Unsubscribe()

	sp.Publish("test.subject", "message")
	time.Sleep(100 * time.Millisecond) // Allow time for potential delivery

	if called {
		t.Error("Handler was called after unsubscribe")
	}
}

func TestClose(t *testing.T) {
	defer goleak.VerifyNone(t)
	sp := NewSubPub()

	// Try to publish after close
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := sp.Close(ctx)
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	err = sp.Publish("test.subject", "message")
	time.Sleep(50 * time.Millisecond)
	if err != context.Canceled {
		t.Errorf("Expected context canceled error, got %v", err)
	}
}

func TestMultipleSubscribers(t *testing.T) {
	defer goleak.VerifyNone(t)
	sp := NewSubPub()
	defer sp.Close(context.Background())

	var (
		wg      sync.WaitGroup
		counter int
		mu      sync.Mutex
	)

	handler := func(msg interface{}) {
		defer wg.Done()
		mu.Lock()
		counter++
		mu.Unlock()
	}

	// Create 5 subscribers
	for i := 0; i < 5; i++ {
		_, err := sp.Subscribe("multi.subject", handler)
		if err != nil {
			t.Fatalf("Subscribe failed: %v", err)
		}
	}

	wg.Add(5)
	sp.Publish("multi.subject", "batch message")
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if counter != 5 {
		t.Errorf("Expected 5 handler calls, got %d", counter)
	}
}

func TestSubjectIsolation(t *testing.T) {
	defer goleak.VerifyNone(t)
	sp := NewSubPub()
	defer sp.Close(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)

	// Subscribe to correct subject
	_, err := sp.Subscribe("correct.subject", func(msg interface{}) {
		defer wg.Done()
		if msg != "right" {
			t.Errorf("Received unexpected message: %v", msg)
		}
	})
	if err != nil {
		t.Fatal(err)
	}

	// Subscribe to wrong subject
	_, err = sp.Subscribe("wrong.subject", func(msg interface{}) {
		t.Error("Handler for wrong subject was called")
	})
	if err != nil {
		t.Fatal(err)
	}

	sp.Publish("correct.subject", "right")
	wg.Wait()
}

func TestConcurrentPublish(t *testing.T) {
	defer goleak.VerifyNone(t)
	sp := NewSubPub()
	defer sp.Close(context.Background())

	var (
		wg      sync.WaitGroup
		counter int
		mu      sync.Mutex
	)

	_, err := sp.Subscribe("concurrent.subject", func(msg interface{}) {
		defer wg.Done()
		mu.Lock()
		counter++
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}

	const numMessages = 100
	wg.Add(numMessages)

	// Concurrent publishes
	for i := 0; i < numMessages; i++ {
		go func() {
			sp.Publish("concurrent.subject", "msg")
		}()
	}

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if counter != numMessages {
		t.Errorf("Expected %d messages, got %d", numMessages, counter)
	}
}
