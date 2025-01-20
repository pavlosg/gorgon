package jrpc

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"time"

	"github.com/pavlosg/gorgon/src/gorgon/log"
)

func Dial(address string, key []byte) (*rpc.Client, error) {
	conn, err := net.DialTimeout("tcp", address, time.Minute)
	if err != nil {
		return nil, err
	}
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()
	buf := NewBufferedStream(conn)
	err = conn.SetDeadline(time.Now().Add(time.Minute))
	if err != nil {
		return nil, err
	}
	err = newAuthenticator(buf, key).authClient()
	if err != nil {
		return nil, err
	}
	err = conn.SetDeadline(time.Time{})
	if err != nil {
		return nil, err
	}
	conn = nil
	return jsonrpc.NewClient(buf), nil
}

func Listen(address string, key []byte) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()
	log.Info("RPC listening on %v", listener.Addr())
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go handleConnection(conn, key)
	}
}

func handleConnection(conn net.Conn, key []byte) {
	defer conn.Close()
	log.Info("RPC accepted %v", conn.RemoteAddr())
	buf := NewBufferedStream(conn)
	err := conn.SetDeadline(time.Now().Add(time.Minute))
	if err != nil {
		return
	}
	err = newAuthenticator(buf, key).authServer()
	if err != nil {
		log.Warning("RPC auth failed %v %v", conn.RemoteAddr(), err)
		return
	}
	log.Info("RPC auth succeeded %v", conn.RemoteAddr())
	err = conn.SetDeadline(time.Time{})
	if err != nil {
		return
	}
	jsonrpc.ServeConn(buf)
}
