package module

import (
	"bufio"
	"cube/internal/builtin"
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

func (s *TCPSocket) Dial(host string, port int) (*TCPSocketConnection, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
	return &TCPSocketConnection{
		conn: &conn,
		AbstractSocketConnection: AbstractSocketConnection{
			reader: bufio.NewReader(conn),
		},
		writer: bufio.NewWriter(conn),
	}, err
}

func (s *TCPSocket) Listen(port int) (*TCPSocketListener, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	s.worker.AddDefer(func() {
		listener.Close()
	})

	return &TCPSocketListener{
		listener: &listener,
	}, err
}

//#region TCP Socket Listener

type TCPSocketListener struct {
	listener *net.Listener
}

func (s *TCPSocketListener) Accept() (*TCPSocketConnection, error) {
	conn, err := (*s.listener).Accept()
	return &TCPSocketConnection{
		&conn,
		AbstractSocketConnection{
			reader: bufio.NewReader(conn),
		},
		bufio.NewWriter(conn),
	}, err
}

//#endregion

//#region TCP Socket Connection

type TCPSocketConnection struct {
	conn *net.Conn
	AbstractSocketConnection
	writer *bufio.Writer
}

func (s *TCPSocketConnection) ReadLine() (builtin.Buffer, error) {
	line, err := s.reader.ReadBytes('\n')

	if err != nil && err != io.EOF {
		return nil, err
	}
	return line, nil
}

func (s *TCPSocketConnection) Write(data []byte) (int, error) {
	count, err := s.writer.Write(data)
	s.writer.Flush()
	return count, err
}

func (s *TCPSocketConnection) Close() {
	if s.conn != nil {
		(*s.conn).Close()
	}
}

//#endregion

//#endregion

//#region UDP

type UDPSocket struct {
	worker Worker
}

func (s *UDPSocket) Dial(host string, port int) (*UDPSocketConnection, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}

	return &UDPSocketConnection{
		conn,
		AbstractSocketConnection{
			reader: bufio.NewReader(conn),
		},
	}, err
}

func (s *UDPSocket) Listen(port int) (*UDPSocketConnection, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	s.worker.AddDefer(func() {
		conn.Close()
	})

	return &UDPSocketConnection{
		conn,
		AbstractSocketConnection{
			reader: bufio.NewReader(conn),
		},
	}, err
}

func (s *UDPSocket) ListenMulticast(host string, port int) (*UDPSocketConnection, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenMulticastUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}

	s.worker.AddDefer(func() {
		conn.Close()
	})

	return &UDPSocketConnection{
		conn,
		AbstractSocketConnection{
			reader: bufio.NewReader(conn),
		},
	}, err
}

//#region UDP Socket Connection

type UDPSocketConnection struct {
	conn *net.UDPConn
	AbstractSocketConnection
}

func (s *UDPSocketConnection) Write(data []byte, host string, port int) (int, error) {
	if host != "" && port != 0 {
		return s.conn.WriteTo(data, &net.UDPAddr{IP: net.ParseIP(host), Port: port})
	}
	return s.conn.Write(data)
}

func (s *UDPSocketConnection) Close() {
	if s.conn != nil {
		(*s.conn).Close()
	}
}

//#endregion

//#endregion

//#region Abstract Socket Connection

type AbstractSocketConnection struct {
	reader *bufio.Reader
}

func (c *AbstractSocketConnection) Read(size int) (builtin.Buffer, error) {
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
		n, err = c.reader.Read(buf)
	}
	if size > 0 {
		buf = make([]byte, size)
		n, err = io.ReadFull(c.reader, buf) // 强制读取 size 大小的字节
	}

	if err != nil && err != io.EOF {
		return nil, err
	}
	return buf[:n], nil
}

//#endregion
