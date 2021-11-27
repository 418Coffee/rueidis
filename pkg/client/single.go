package client

import (
	"time"

	"github.com/rueian/rueidis/internal/cmds"
	"github.com/rueian/rueidis/internal/proto"
	"github.com/rueian/rueidis/pkg/conn"
	"github.com/rueian/rueidis/pkg/om"
)

type SingleClientOption struct {
	Address    string
	ConnOption conn.Option
}

type SingleClient struct {
	Cmd  *cmds.Builder
	conn *conn.Conn
}

func NewSingleClient(option SingleClientOption) (*SingleClient, error) {
	c := conn.NewConn(option.Address, option.ConnOption)
	if err := c.Dialable(); err != nil {
		return nil, err
	}
	return &SingleClient{Cmd: cmds.NewBuilder(), conn: c}, nil
}

func (c *SingleClient) Info() map[string]proto.Message {
	return c.conn.Info()
}

func (c *SingleClient) Do(cmd cmds.Completed) (resp proto.Result) {
	resp = c.conn.Do(cmd)
	c.Cmd.Put(cmd.Commands())
	return resp
}

func (c *SingleClient) DoCache(cmd cmds.Cacheable, ttl time.Duration) (resp proto.Result) {
	resp = c.conn.DoCache(cmd, ttl)
	c.Cmd.Put(cmd.Commands())
	return resp
}

func (c *SingleClient) DedicatedWire(fn func(DedicatedSingleClient) error) (err error) {
	wire := c.conn.Acquire()
	err = fn(DedicatedSingleClient{cmd: c.Cmd, wire: wire})
	c.conn.Store(wire)
	return err
}

func (c *SingleClient) NewHashRepository(prefix string, schema interface{}) *om.HashRepository {
	return om.NewHashRepository(
		prefix,
		schema,
		func(key string, fields map[string]string) (ver int64, err error) {
			cmd := c.Cmd.Hset().Key(key).FieldValue()
			for f, v := range fields {
				cmd = cmd.FieldValue(f, v)
			}
			_, err = c.Do(cmd.Build()).ToInt64()
			return
		},
		func(key string) (map[string]proto.Message, error) {
			return c.Do(c.Cmd.Hgetall().Key(key).Build()).ToMap()
		},
		func(key string, ttl time.Duration) (map[string]proto.Message, error) {
			return c.DoCache(c.Cmd.Hgetall().Key(key).Cache(), ttl).ToMap()
		})
}

func (c *SingleClient) Close() {
	c.conn.Close()
}

type DedicatedSingleClient struct {
	cmd  *cmds.Builder
	wire conn.Wire
}

func (c *DedicatedSingleClient) Do(cmd cmds.Completed) (resp proto.Result) {
	resp = c.wire.Do(cmd)
	c.cmd.Put(cmd.Commands())
	return resp
}

func (c *DedicatedSingleClient) DoMulti(multi ...cmds.Completed) (resp []proto.Result) {
	if len(multi) == 0 {
		return nil
	}
	resp = c.wire.DoMulti(multi...)
	for _, cmd := range multi {
		c.cmd.Put(cmd.Commands())
	}
	return resp
}