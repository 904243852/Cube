package module

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
)

func init() {
	register("socket", func(worker Worker, db Db) interface{} {
		return func(protocol string) (interface{}, error) {
			if protocol == "tcp" {
				return &TCPSocket{
					worker,
				}, nil
			}
			if protocol == "udp" {
				return &UDPSocket{
					worker,
				}, nil
			}
			return nil, errors.New("unsupported protocol: must be tcp or udp")
		}
	})
}

//#region TCP

type TCPSocket struct {
	worker Worker
}

func (s *TCPSocket) Dial(host string, port int) (*SocketConnection, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	return &SocketConnection{
		conn:   &conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}, err
}

func (s *TCPSocket) Listen(port int) (*TCPSocketListener, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	s.worker.AddHandle(&listener)

	return &TCPSocketListener{
		listener: &listener,
	}, err
}

//#region TCP Socket Listener

type TCPSocketListener struct {
	listener *net.Listener
}

func (s *TCPSocketListener) Accept() (*SocketConnection, error) {
	conn, err := (*s.listener).Accept()
	return &SocketConnection{
		conn:   &conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}, err
}

//#endregion

//#endregion

//#region UDP

type UDPSocket struct {
	worker Worker
}

func (s *UDPSocket) Dial(host string, port int) (*SocketConnection, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}

	return &SocketConnection{
		conn2:  conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}, err
}

func (s *UDPSocket) Listen(port int) (*SocketConnection, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	s.worker.AddHandle(conn)

	return &SocketConnection{
		conn2:  conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}, err
}

func (s *UDPSocket) ListenMulticast(host string, port int) (*SocketConnection, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}

	s.worker.AddHandle(conn)

	return &SocketConnection{
		conn2:  conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}, err
}

//#endregion

//#region Socket Connection

type SocketConnection struct {
	conn   *net.Conn
	conn2  *net.UDPConn
	reader *bufio.Reader
	writer *bufio.Writer
}

func (s *SocketConnection) Read(size int) ([]byte, error) {
	var (
		buf []byte
		n   int
		err error
	)

	if size < 0 {
		return nil, errors.New("invalid size: must be greater than or equal to 0")
	}
	if size == 0 {
		buf = make([]byte, 65535) // 默认最大缓存长度 65535 字节
		n, err = s.reader.Read(buf)
	}
	if size > 0 {
		buf = make([]byte, size)
		n, err = io.ReadFull(s.reader, buf) // 强制读取 size 大小的字节
	}

	if err != nil && err != io.EOF {
		return nil, err
	}
	return buf[:n], nil
}

func (s *SocketConnection) ReadLine() ([]byte, error) {
	line, err := s.reader.ReadBytes('\n')

	if err != nil && err != io.EOF {
		return nil, err
	}
	return line, nil
}

func (s *SocketConnection) Write(data []byte) (int, error) {
	count, err := s.writer.Write(data)
	s.writer.Flush()
	return count, err
}

func (s *SocketConnection) Close() {
	if s.conn != nil {
		(*s.conn).Close()
	}
	if s.conn2 != nil {
		(*s.conn2).Close()
	}
}

//#endregion
