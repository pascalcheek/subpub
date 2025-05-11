package subpub

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPubSub(t *testing.T) {
	bus := New()

	t.Run("basic pubsub", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)

		sub, err := bus.Subscribe("test", func(msg interface{}) {
			assert.Equal(t, "hello", msg)
			wg.Done()
		})
		assert.NoError(t, err)

		err = bus.Publish("test", "hello")
		assert.NoError(t, err)

		wg.Wait()
		sub.Unsubscribe()
	})

	t.Run("unsubscribe", func(t *testing.T) {
		called := false
		sub, _ := bus.Subscribe("test", func(msg interface{}) {
			called = true
		})

		sub.Unsubscribe()
		bus.Publish("test", "hello")
		time.Sleep(100 * time.Millisecond)
		assert.False(t, called)
	})

	t.Run("shutdown", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := bus.Close(ctx)
		assert.NoError(t, err)

		err = bus.Publish("test", "after close")
		assert.Error(t, err)
	})
}
