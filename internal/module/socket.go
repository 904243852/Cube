package module

import (
	"bufio"
	"fmt"
	"io"
	"net"
)

func init() {
	register("socket", func(worker Worker, db Db) interface{} {
		return &Socket{worker}
	})
}

type Socket struct {
	worker Worker
}
type SocketListener struct {
	listener *net.Listener
}

func (s *Socket) Listen(protocol string, port int) (*SocketListener, error) {
	listener, err := net.Listen(protocol, fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	s.worker.AddHandle(&listener)
	return &SocketListener{
		listener: &listener,
	}, err
}

func (s *Socket) Dial(protocol string, host string, port int) (*SocketConn, error) {
	conn, err := net.Dial(protocol, fmt.Sprintf("%s:%d", host, port))
	return &SocketConn{
		conn:   &conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}, err
}

type SocketConn struct {
	conn   *net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

func (s *SocketListener) Accept() (*SocketConn, error) {
	conn, err := (*s.listener).Accept()
	return &SocketConn{
		conn:   &conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}, err
}

func (s *SocketConn) ReadLine() ([]byte, error) {
	line, err := s.reader.ReadBytes('\n')
	if err == io.EOF {
		return nil, nil
	}
	return line, err
}

func (s *SocketConn) Write(data []byte) (int, error) {
	count, err := s.writer.Write(data)
	s.writer.Flush()
	return count, err
}

func (s *SocketConn) Close() {
	(*s.conn).Close()
}
