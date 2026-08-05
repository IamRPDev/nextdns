package proxy

import (
	"errors"
	"net"
	"sync"
	"testing"
	"time"
)

func TestServeTCPConn_IdleReadTimeout(t *testing.T) {
	prev := tcpClientReadTimeout
	tcpClientReadTimeout = 50 * time.Millisecond
	t.Cleanup(func() {
		tcpClientReadTimeout = prev
	})

	server, client := net.Pipe()
	defer client.Close()

	bpool := &sync.Pool{
		New: func() any {
			return new(tcpBuf)
		},
	}
	inflightRequests := make(chan struct{}, 1)
	errC := make(chan error, 1)

	go func() {
		errC <- (Proxy{}).serveTCPConn(server, inflightRequests, bpool)
	}()

	select {
	case err := <-errC:
		if err == nil {
			t.Fatal("serveTCPConn() err = nil, want timeout error")
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("serveTCPConn() did not time out idle client")
	}

	if got := len(inflightRequests); got != 0 {
		t.Fatalf("inflightRequests len = %d, want 0", got)
	}
}

func TestServeTCPConn_FullLimitDoesNotBlock(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()

	bpool := &sync.Pool{New: func() any { return new(tcpBuf) }}
	inflightRequests := make(chan struct{}, 1)
	inflightRequests <- struct{}{}
	errC := make(chan error, 1)
	go func() {
		errC <- (Proxy{}).serveTCPConn(server, inflightRequests, bpool)
	}()

	select {
	case err := <-errC:
		if !errors.Is(err, errTooManyInflightRequests) {
			t.Fatalf("serveTCPConn() error = %v, want inflight limit error", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("serveTCPConn() blocked on a full inflight limit")
	}
}
