package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nsqio/go-nsq"

	"github.com/Sirupsen/logrus"
	"github.com/chennqqi/goutils/closeevent"
)

var (
	httpAddress      = flag.String("http-address", "127.0.0.1:8080", "<addr>:<port> to listen on for HTTP clients")
	maxInFlight      = flag.Int("max-in-flight", 100, "max number of messages to allow in flight")
	lookupdHTTPAddrs = StringArray{}
	nsqdTCPAddrs     = StringArray{}
	timeout          = flag.Int("timeout", 10, "return within N seconds if maxMessages not reached")
	maxMessages      = flag.Int("max-messages", 1, "return if got N messages in a single poll")
	topic            = flag.String("topic", "", "set nsq topic")
	channel          = flag.String("channel", "", "set nsq group")
)

func init() {
	flag.Var(&lookupdHTTPAddrs, "lookupd-http-address", "lookupd HTTP address (may be given multiple times)")
	flag.Var(&nsqdTCPAddrs, "lookupd-http-address", "lookupd HTTP address (may be given multiple times)")
}

type HttpContext struct {
	c       *gin.Context
	stop    chan struct{}
	want    int
	count   int
	start   time.Time
	timeout time.Duration
	buf     *bytes.Buffer
}

type WebServer struct {
	consumer *nsq.Consumer
	//TODO:
	queue *Queue
}

func GetInt(c *gin.Context, key string) int {
	var r int
	limit := c.Query(key)
	fmt.Sscanf(limit, "%d", &r)
	return r
}

func GetDuration(c *gin.Context, key string) (time.Duration, error) {
	to := c.Query(key)
	return time.ParseDuration(to)
}

func (w *WebServer) sub(c *gin.Context) {
	stopChan := make(chan struct{})
	q := w.queue

	limit := GetInt(c, "limit")
	timeout, err := GetDuration(c, "timeout")
	if err != nil {
		timeout, _ = time.ParseDuration("1s")
	}
	ctx := &HttpContext{
		c, stopChan, limit, 0, time.Now(), timeout, bytes.NewBuffer(nil),
	}

	ctx.buf.WriteByte('[')

	q.PushBack(ctx)
	<-stopChan

	if ctx.count == 0 {
		c.Status(http.StatusNotModified)
		return
	}

	ctx.buf.WriteByte(']')
	c.Data(200, "hex", ctx.buf.Bytes())
}

func (w *WebServer) stat(c *gin.Context) {
	consumer := w.consumer
	st := consumer.Stats()
	c.JSON(200, st)
}

func (w *WebServer) Init() error {
	nsqConfig := nsq.NewConfig()
	nsqConfig.MaxInFlight = *maxInFlight
	nsqConfig.WriteTimeout = 3 * time.Second
	nsqConfig.DialTimeout = 4 * time.Second

	consumer, err := nsq.NewConsumer(*topic, *channel, nsqConfig)
	if err != nil {
		logrus.Error("[main:NsqGroup.Open] NewConsumer error ", err)
		return err
	}

	w.consumer = consumer
	consumer.AddHandler(w)

	err = connectToNSQAndLookupd(consumer, lookupdHTTPAddrs, nsqdTCPAddrs)
	if err != nil {
		logrus.Error("[main:NsqGroup.Open] ConnectToNSQLookupds error ", err)
		return err
	}
	if false { // maybe no data
		stats := consumer.Stats()
		if stats.Connections == 0 {
			logrus.Error("[main:NsqGroup.Open] consumer.Stats report 0 connections (should be > 0)")
			return errors.New("stats report 0 connections (should be > 0)")
		}
	}

	w.queue = NewQueue()
	return nil
}

func (w *WebServer) shutdown() {
}

func (w *WebServer) run() {
	r := gin.Default()
	r.GET("/sub", w.sub)
	r.GET("/stat", w.stat)

	srv := &http.Server{
		Addr:    *httpAddress,
		Handler: r,
	}

	// service connections
	if err := srv.ListenAndServe(); err != nil {
		log.Printf("listen: %s\n", err)
	}
}

func (w *WebServer) HandleMessage(m *nsq.Message) error {
	q := w.queue
	v := q.PopFront()
	if v == nil {
		return errors.New("NOT READER")
	}
	h := v.(*HttpContext)
	if h.count > 0 {
		h.buf.WriteByte(',')
	}
	h.buf.Write(m.Body)
	h.count++
	if h.count == h.want || time.Now().Sub(h.start) >= h.timeout {
		close(h.stop)
		return nil
	}
	//push to front for next
	q.PushFront(h)
	return nil
}

func connectToNSQAndLookupd(r *nsq.Consumer, nsqAddrs []string, lookupd []string) error {
	for _, addrString := range nsqAddrs {
		err := r.ConnectToNSQD(addrString)
		if err != nil {
			return err
		}
	}

	for _, addrString := range lookupd {
		log.Printf("lookupd addr %s", addrString)
		err := r.ConnectToNSQLookupd(addrString)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	flag.Parse()

	if *maxInFlight <= 0 {
		log.Fatalf("--max-in-flight must be > 0")
	}

	if len(lookupdHTTPAddrs) == 0 {
		log.Fatalf("--lookupd-http-address required.")
	}

	var w WebServer
	w.Init()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		w.run()
		wg.Done()
	}()

	//graceful shutdown
	closeevent.Wait(func(sig os.Signal) {
		w.shutdown()
		wg.Wait()
	})
}
