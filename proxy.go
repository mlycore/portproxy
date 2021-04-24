package main

import (
	"io"
	"github.com/mlycore/log"
	"io/ioutil"
	"net"
	"strings"
	"sync/atomic"
	"time"
)

type Proxy struct {
	bind, backend *net.TCPAddr
	sessionsCount int32
	pool          *recycler
}

func New(bind, backend string, size uint32) *Proxy {
	a1, err := net.ResolveTCPAddr("tcp", bind)
	if err != nil {
		log.Fatalln("resolve bind error:", err)
	}

	a2, err := net.ResolveTCPAddr("tcp", backend)
	if err != nil {
		log.Fatalln("resolve backend error:", err)
	}

	return &Proxy{
		bind:          a1,
		backend:       a2,
		sessionsCount: 0,
		pool:          NewRecycler(size),
	}
}

func (t *Proxy) pipe(dst, src *Conn, c chan int64, tag string) {
	defer func() {
		dst.CloseWrite()
		dst.CloseRead()
	}()
	// local->proxy
	if strings.EqualFold(tag, "send") {
		proxyLog(src, dst)
		c <- 0
	} else {
	// proxy->local
		n, err := io.Copy(dst, src)
		data, err := ioutil.ReadAll(src)
		log.Errorf("receive read dst: %d, %s", n, string(data))
		if err != nil {
			log.Errorln(err)
		}
		c <- n
	}
}

func (t *Proxy) transport(local net.Conn) {
	start := time.Now()
	proxy, err := net.DialTCP("tcp", nil, t.backend)
	if err != nil {
		log.Errorln(err)
		return
	}
	connectTime := time.Now().Sub(start)
	log.Infof("proxy: %s ==> %s", proxy.LocalAddr().String(),
		proxy.RemoteAddr().String())
	start = time.Now()
	readChan := make(chan int64)
	writeChan := make(chan int64)
	var readBytes, writeBytes int64

	atomic.AddInt32(&t.sessionsCount, 1)
	var localConn, proxyConn *Conn
	localConn = NewConn(local, t.pool)
	proxyConn = NewConn(proxy, t.pool)

	go t.pipe(proxyConn, localConn, writeChan, "send") //localConn -> proxyConn
	go t.pipe(localConn, proxyConn, readChan, "receive") //proxyConn -> localConn

	// blocked here, waiting for all communication finished
	readBytes = <-readChan //once proxy->local, it's been run
	log.Errorf("readBytes: %d", readBytes)
	writeBytes = <-writeChan
	log.Errorf("writeBytes: %d", writeBytes)
	transferTime := time.Now().Sub(start)
	log.Fatalf("r: %d w:%d ct:%.3f t:%.3f [#%d]", readBytes, writeBytes,
		connectTime.Seconds(), transferTime.Seconds(), t.sessionsCount)
	atomic.AddInt32(&t.sessionsCount, -1)
}

func (t *Proxy) Start() {
	ln, err := net.ListenTCP("tcp", t.bind)
	if err != nil {
		log.Fatalln(err)
	}

	defer ln.Close()
	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			log.Errorf("accept:", err)
			continue
		}
		log.Infof("client: %s ==> %s", conn.RemoteAddr().String(),
			conn.LocalAddr().String())
		go t.transport(conn)
	}
}
