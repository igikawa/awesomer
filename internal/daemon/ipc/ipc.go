package ipc

import (
	"awesomeProject/internal/daemon/info"
	"context"
	"errors"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
)

const SocketPath = "/run/awesomer.sock"

type PidArgs struct {
	PID int
}

type ToggleReply struct {
	InJail bool
}

type NoopArgs struct {
	Dummy bool
}

type Controller interface {
	ToggleProcessJail(pid int) (bool, error)
}

type JailRPC struct {
	State      info.JailState
	Controller Controller
}

func (j *JailRPC) Ping(_ *NoopArgs, reply *bool) error {
	*reply = true
	return nil
}

func (j *JailRPC) InJail(args *PidArgs, reply *bool) error {
	*reply = j.State.InJail(args.PID)
	return nil
}

func (j *JailRPC) SetJail(args *PidArgs, reply *bool) error {
	j.State.SetJail(args.PID)
	*reply = true
	return nil
}

func (j *JailRPC) DeleteFromJail(args *PidArgs, reply *bool) error {
	j.State.DeleteFromJail(args.PID)
	*reply = true
	return nil
}

func (j *JailRPC) PIDs(_ *NoopArgs, reply *[]int) error {
	*reply = j.State.PIDs()
	return nil
}

func (j *JailRPC) ToggleProcessJail(args *PidArgs, reply *ToggleReply) error {
	if j.Controller == nil {
		return errors.New("daemon control is unavailable")
	}

	inJail, err := j.Controller.ToggleProcessJail(args.PID)
	if err != nil {
		return err
	}
	reply.InJail = inJail
	return nil
}

type Server struct {
	listener net.Listener
	path     string
}

func Start(ctx context.Context, path string, state info.JailState, controller Controller) (*Server, error) {
	if state == nil {
		return nil, errors.New("ipc state is nil")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create socket dir: %w", err)
	}
	if err := removeStaleSocket(path); err != nil {
		return nil, err
	}

	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", path, err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		_ = listener.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("chmod socket %s: %w", path, err)
	}

	server := rpc.NewServer()
	if err := server.RegisterName("JailState", &JailRPC{State: state, Controller: controller}); err != nil {
		_ = listener.Close()
		_ = os.Remove(path)
		return nil, fmt.Errorf("register jail rpc: %w", err)
	}

	s := &Server{listener: listener, path: path}
	go s.serve(ctx, server)
	return s, nil
}

func (s *Server) Close() error {
	if s == nil {
		return nil
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
	if s.path != "" {
		_ = os.Remove(s.path)
	}
	return nil
}

func (s *Server) serve(ctx context.Context, server *rpc.Server) {
	go func() {
		<-ctx.Done()
		_ = s.Close()
	}()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if ctx.Err() != nil || isClosedConnError(err) {
				return
			}
			continue
		}

		go server.ServeConn(conn)
	}
}

type Client struct {
	path string
}

func NewClient(path string) *Client {
	return &Client{path: path}
}

func (c *Client) Ping() error {
	var ok bool
	return c.call("JailState.Ping", &NoopArgs{}, &ok)
}

func (c *Client) InJail(pid int) bool {
	var inJail bool
	if err := c.call("JailState.InJail", &PidArgs{PID: pid}, &inJail); err != nil {
		return false
	}
	return inJail
}

func (c *Client) SetJail(pid int) {
	var ok bool
	_ = c.call("JailState.SetJail", &PidArgs{PID: pid}, &ok)
}

func (c *Client) DeleteFromJail(pid int) {
	var ok bool
	_ = c.call("JailState.DeleteFromJail", &PidArgs{PID: pid}, &ok)
}

func (c *Client) PIDs() []int {
	var pids []int
	if err := c.call("JailState.PIDs", &NoopArgs{}, &pids); err != nil {
		return nil
	}
	return pids
}

func (c *Client) ToggleProcessJail(pid int) (bool, error) {
	var reply ToggleReply
	if err := c.call("JailState.ToggleProcessJail", &PidArgs{PID: pid}, &reply); err != nil {
		return false, err
	}
	return reply.InJail, nil
}

func IsAvailable(path string) bool {
	client := NewClient(path)
	return client.Ping() == nil
}

func (c *Client) call(method string, args any, reply any) error {
	conn, err := rpc.Dial("unix", c.path)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	return conn.Call(method, args, reply)
}

func removeStaleSocket(path string) error {
	if !socketExists(path) {
		return nil
	}
	if IsAvailable(path) {
		return fmt.Errorf("daemon socket already in use: %s", path)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove stale socket %s: %w", path, err)
	}
	return nil
}

func socketExists(path string) bool {
	inf, err := os.Stat(path)
	if err != nil {
		return false
	}
	return inf.Mode()&os.ModeSocket != 0 || strings.HasSuffix(inf.Mode().String(), "S")
}

func isClosedConnError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "use of closed network connection")
}
