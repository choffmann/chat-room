package chat

import (
	"testing"
)

func TestClientCloseSend(t *testing.T) {
	client := &Client{
		send:   make(chan []byte, 256),
		closed: false,
		logger: testLogger(),
	}

	client.CloseSend()

	_, ok := <-client.send
	if ok {
		t.Error("send channel should be closed")
	}

	// Second close should not panic
	client.CloseSend()
}
