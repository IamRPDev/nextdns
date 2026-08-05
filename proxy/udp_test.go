package proxy

import (
	"net"
	"testing"
	"time"
)

func TestServeUDP_FullLimitReturnsServerFailure(t *testing.T) {
	server, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatalf("ListenUDP() error = %v", err)
	}
	t.Cleanup(func() { server.Close() })
	client, err := net.DialUDP("udp", nil, server.LocalAddr().(*net.UDPAddr))
	if err != nil {
		server.Close()
		t.Fatalf("DialUDP() error = %v", err)
	}
	defer client.Close()

	inflightRequests := make(chan struct{}, 1)
	inflightRequests <- struct{}{}
	errC := make(chan error, 1)
	go func() {
		errC <- (Proxy{}).serveUDP(server, inflightRequests)
	}()

	query := []byte{
		0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x04, 't', 'e', 's', 't', 0x00, 0x00, 0x01, 0x00, 0x01,
	}
	if _, err := client.Write(query); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := client.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	response := make([]byte, 512)
	n, err := client.Read(response)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if n < 4 || response[3]&0x0f != 2 {
		t.Fatalf("response = %v, want SERVFAIL", response[:n])
	}

	if err := server.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case <-errC:
	case <-time.After(time.Second):
		t.Fatal("serveUDP() did not stop while the inflight limit was full")
	}
}
