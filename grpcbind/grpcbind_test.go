package grpcbind

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
)

func TestParse(t *testing.T) {

	for _, tc := range []struct {
		in          string
		wantNetwork string
		wantAddr    string
	}{{
		in:          ":8080",
		wantNetwork: "tcp",
		wantAddr:    ":8080",
	}, {
		in:          "/path/to/socket",
		wantNetwork: "unix",
		wantAddr:    "/path/to/socket",
	}, {
		in:          "unix:///path/to/socket",
		wantNetwork: "unix",
		wantAddr:    "/path/to/socket",
	}, {
		in:          "unix://../relative/path/to/socket",
		wantNetwork: "unix",
		wantAddr:    "../relative/path/to/socket",
	}, {
		in:          "tcp://:8080",
		wantNetwork: "tcp",
		wantAddr:    ":8080",
	}, {
		in:          "tcp://:0",
		wantNetwork: "tcp",
		wantAddr:    ":0",
	}, {
		in:          "tcp://127.0.0.1:8080",
		wantNetwork: "tcp",
		wantAddr:    "127.0.0.1:8080",
	}} {
		network, addr, err := urlToListener(tc.in)
		if err != nil {
			t.Errorf("urlToListener(%q) threw %q", tc.in, err)
			continue
		}
		if network != tc.wantNetwork || addr != tc.wantAddr {
			t.Errorf("urlToListener(%q) = %q, %q, nil; want %q, %q, nil", tc.in, network, addr, tc.wantNetwork, tc.wantAddr)
		}

	}

	for _, tc := range []string{
		"",
		"foo",
		"s3://:8080",
	} {
		n, a, err := urlToListener(tc)
		if err == nil {
			t.Errorf("urlToListener(%q) did not throw, got %s %q", tc, n, a)
		}
	}
}

type mockServer struct {
	onStop func()
	accept func(net.Conn)
}

func (s *mockServer) Serve(lis net.Listener) error {
	for {
		conn, err := lis.Accept()
		if err != nil {
			return err
		}
		s.accept(conn)
	}
}

func (s *mockServer) GracefulStop() {
	s.onStop()

}

func TestHangup(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chStop := make(chan struct{})
	chConns := make(chan net.Conn, 1)
	server := &mockServer{
		onStop: func() {
			chStop <- struct{}{}
		},
		accept: func(conn net.Conn) {
			chConns <- conn
		},
	}

	tmp, err := os.MkdirTemp("", "")
	if err != nil {
		t.Errorf("os.MkdirTemp() threw %q", err)
	}
	defer os.RemoveAll(tmp)

	bind := fmt.Sprintf("unix://%s/test.sock", tmp)
	lis, err := Listen(ctx, bind)
	if err != nil {
		t.Errorf("ListenAndServe(%q) threw %q", bind, err)
		return
	}

	go func() {
		err := Serve(ctx, server, lis)
		if err != nil {
			t.Errorf("Serve(%q) threw %q", bind, err)
		}
	}()

	conn1, err := net.Dial("unix", tmp+"/test.sock")
	if err != nil {
		t.Errorf("net.Dial(%q) threw %q", bind, err)
		return
	}

	<-chConns

	conn2, err := net.Dial("unix", tmp+"/test.sock")
	if err != nil {
		t.Errorf("net.Dial(%q) threw %q", bind, err)
		return
	}

	<-chConns

	cancel()

	conn1.Close()
	conn2.Close()

	<-chStop
	t.Log("graceful stop")

}
