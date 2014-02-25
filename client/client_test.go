package client_test

import (
	"github.com/nelhage/livegrep/client"
	. "launchpad.net/gocheck"
	"net"
	"strings"
	"testing"
)

// We assume a codesearch running on localhost:9999. This could be
// improved.

func Test(t *testing.T) { TestingT(t) }

type ClientSuite struct {
	client client.Client
}

var _ = Suite(&ClientSuite{})

func (s *ClientSuite) SetUpTest(c *C) {
	var err error
	s.client, err = client.Dial("tcp", "localhost:9999")
	if err != nil {
		panic(err.Error())
	}
}

func (s *ClientSuite) TestQuery(c *C) {
	search, err := s.client.Query(&client.Query{".", "", ""})
	c.Assert(err, IsNil)
	var n int
	for r := range search.Results() {
		n++
		c.Assert(r.Line, Not(Equals), "")
	}
	c.Assert(n, Not(Equals), 0)
	st, e := search.Close()
	c.Assert(st, Not(IsNil))
	c.Assert(e, IsNil)
}

func (s *ClientSuite) TestTwoQueries(c *C) {
	search, err := s.client.Query(&client.Query{".", "", ""})
	c.Assert(err, IsNil)
	_, err = search.Close()
	c.Assert(err, IsNil)

	search, err = s.client.Query(&client.Query{".", "", ""})
	c.Assert(err, IsNil)
	n := 0
	for _ = range search.Results() {
		n++
	}
	_, err = search.Close()
	if err != nil {
		c.Fatalf("Unexpected error: %s", err.Error())
	}
	c.Assert(n, Not(Equals), 0)
}

func (s *ClientSuite) TestBadRegex(c *C) {
	search, err := s.client.Query(&client.Query{"(", "", ""})
	c.Assert(err, IsNil)
	for _ = range search.Results() {
		c.Fatal("Got back a result from an erroneous query!")
	}
	st, e := search.Close()
	c.Assert(st, IsNil)
	if e == nil {
		c.Fatal("Didn't get back an error")
	}
	if q, ok := e.(client.QueryError); ok {
		c.Assert(q.Query.Line, Equals, "(")
		if strings.HasPrefix(q.Err, "FATAL") {
			c.Errorf("Error includes FATAL prefix: %s", q.Err)
		}
	} else {
		c.Fatalf("Error %v wasn't a QueryError", e)
	}
}

func shutdownClient(addr string, ready chan<- bool) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err.Error())
	}
	defer ln.Close()
	ready <- true
	conn, err := ln.Accept()
	if err != nil {
		panic(err.Error())
	}
	conn.Write([]byte("READY {}\n"))
	conn.Close()
}

func (s *ClientSuite) TestShutdown(c *C) {
	ready := make(chan bool, 1)
	go shutdownClient("127.0.0.1:10999", ready)
	<-ready

	cl, err := client.Dial("tcp", "127.0.0.1:10999")
	c.Assert(err, IsNil)

	search, err := cl.Query(&client.Query{Line: "l"})
	c.Assert(err, IsNil)
	c.Assert(search, Not(IsNil))

	results := search.Results()
	c.Assert(results, Not(IsNil))
	for r := range results {
		c.Errorf("Got a result back: %+v", r)
	}
	st, err := search.Close()
	c.Assert(st, IsNil)
	c.Assert(err, Not(IsNil))

	search, err = cl.Query(&client.Query{Line: "l"})
	c.Assert(err, Not(IsNil))
	c.Assert(search, IsNil)
}