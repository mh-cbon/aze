package main

import (
	"context"
	_ "expvar"
	"flag"
	"fmt"
	"io"
	stdlog "log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/juju/ratelimit"
	"github.com/oxtoacart/bpool"
)

var (
	log = logAPI{prefix: "[aze] "}
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var opts = struct {
		quiet   bool
		timeout time.Duration
		monitor string
		global  bool
	}{}

	flag.DurationVar(&opts.timeout, "t", time.Second*60, "timeout of the conenction")
	flag.BoolVar(&opts.quiet, "q", false, "stfu")
	flag.BoolVar(&opts.global, "g", false, "global rate limiting")
	flag.StringVar(&opts.monitor, "monitor", ":", "http monitor address")

	flag.Parse()

	go func() {
		if err := http.ListenAndServe(opts.monitor, nil); err != nil {
			log.Fatal("monitoring server could not listen, err=%v", err)
		}
	}()

	log.SetQuiet(opts.quiet)

	// parse <dst> <src> <cap>
	args := flag.Args()
	delta := 1.2

	log.prefix = fmt.Sprintf("[aze %v] ", opts.monitor)

	{
		if args[0] == "gen" {
			log.prefix = fmt.Sprintf("[aze gen %v] ", opts.monitor)
			if len(args) < 3 {
				log.Fatal("invalid command line, expected aze gen <size> <sample> <cap>")
			}
			var size int64
			{
				c, err := bytefmt.ToBytes(args[1])
				if err != nil {
					log.Fatal("failed to parse %q, err=%v", args[1], err)
				}
				size = int64(c)
			}

			speed := int64(1024 * 5)
			if len(args) > 3 {
				c, err := bytefmt.ToBytes(args[3])
				if err != nil {
					log.Fatal("failed to parse %q, err=%v", args[1], err)
				}
				speed = int64(float64(c) * delta)
			}

			var i int64
			sampleLen := int64(len(args[2]))
			sample := []byte(args[2])
			blockSample := make([]byte, 0, int(float64(speed)*delta))
			for i = 0; i < speed; i += sampleLen {
				blockSample = append(blockSample, sample...)
			}
			blockSampleLen := int64(len(blockSample))
			n := 0
			for i = 0; i < size; i += blockSampleLen {
				var y int
				var err error
				if i+blockSampleLen > size {
					y, err = os.Stdout.Write(blockSample[:(i+blockSampleLen)-size])
				} else {
					y, err = os.Stdout.Write(blockSample)
				}
				if err != nil {
					log.Fatal("write error %v", err)
				}
				n += y
			}
			log.Print("written %v", bytefmt.ByteSize(uint64(n)))
			return
		}
	}

	if len(args) < 3 {
		log.Fatal("invalid command line, expected aze <dst> <src> <cap>")
	}

	dstAddr := args[0]
	srcAddr := args[1]
	cap := args[2]

	var capBytes uint64
	{
		b, err := bytefmt.ToBytes(cap)
		if err != nil {
			log.Fatal("failed to parse capacity %q, err=%v", cap, err)
		}
		capBytes = b
	}

	var globalLimiter *ratelimit.Bucket
	if opts.global {
		globalLimiter = ratelimit.NewBucketWithRate(float64(capBytes), int64(float64(capBytes)*delta))
	}

	dstProto := "tcp"
	srcProto := "tcp"

	proto := regexp.MustCompile(`(i)?(tcp|tcp6|tcp4|udp|udp6|udp4)://`)

	if proto.MatchString(dstAddr) {
		dstProto = proto.FindString(dstAddr)
		dstProto = strings.TrimSuffix(dstProto, "://")
		dstAddr = proto.ReplaceAllString(dstAddr, "")
	}

	if proto.MatchString(srcAddr) {
		srcProto = proto.FindString(srcAddr)
		srcProto = strings.TrimSuffix(srcProto, "://")
		srcAddr = proto.ReplaceAllString(srcAddr, "")
	}

	l, err := net.Listen(srcProto, srcAddr)
	if err != nil {
		log.Fatal("failed to listen address %v%q, err=%v", srcProto, srcAddr, err)
	}

	go func() {
		bufpool := bpool.NewBytePool(48, int(float64(capBytes)*delta))
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			var srcConn net.Conn
			{
				c, err := l.Accept()
				if err != nil {
					log.Error("accept error %v", err)
					continue
				}
				defer c.Close()
				srcConn = &idleTimeoutConn{Conn: c, timeout: opts.timeout}
			}
			go func(srcConn net.Conn) {
				var dstConn net.Conn
				{
					d, err := net.Dial(dstProto, dstAddr)
					if err != nil {
						log.Error("dial error %v", err)
						return
					}
					defer d.Close()
					dstConn = &idleTimeoutConn{Conn: d, timeout: opts.timeout}
				}

				var dst io.Writer = dstConn
				var src io.Reader = srcConn
				{

					bucket := globalLimiter
					if !opts.global {
						bucket = ratelimit.NewBucketWithRate(float64(capBytes), int64(capBytes*2))
					}

					dst = ratelimit.Writer(dst, bucket)
					// src = ratelimit.Reader(src, bucket)
				}

				start := time.Now()
				buf := bufpool.Get()
				defer bufpool.Put(buf)
				n, err := io.CopyBuffer(dst, src, buf)
				if err != nil {
					log.Print("[ERR] %v -> %v %v",
						srcConn.RemoteAddr(), dstConn.RemoteAddr(), err,
					)
					return
				}
				elapsed := time.Since(start)
				copied := bytefmt.ByteSize(uint64(n))
				speed := ""
				if x := elapsed.Seconds(); x > 0 {
					speed = bytefmt.ByteSize(uint64(n / int64(x)))
				}
				log.Print("%v -> %v copied %v - %v - %v/s",
					srcConn.RemoteAddr(), dstConn.RemoteAddr(),
					copied, elapsed, speed,
				)
			}(srcConn)
		}
	}()

	<-cancelNotifier()
	cancel()

}

func cancelNotifier() chan os.Signal {
	sig := make(chan os.Signal, 10)
	signal.Notify(sig, os.Interrupt, os.Kill)
	return sig
}

type idleTimeoutConn struct {
	net.Conn
	timeout time.Duration
}

func (i idleTimeoutConn) Read(buf []byte) (n int, err error) {
	i.Conn.SetDeadline(time.Now().Add(i.timeout))
	n, err = i.Conn.Read(buf)
	// i.Conn.SetDeadline(time.Now().Add(i.timeout))
	return
}

func (i idleTimeoutConn) Write(buf []byte) (n int, err error) {
	i.Conn.SetDeadline(time.Now().Add(i.timeout))
	n, err = i.Conn.Write(buf)
	// i.Conn.SetDeadline(time.Now().Add(i.timeout))
	return
}

type logAPI struct {
	stfu   bool
	prefix string
}

func (l *logAPI) SetQuiet(stfu bool) {
	l.stfu = stfu
}

func (l logAPI) Print(f string, args ...interface{}) {
	if l.stfu {
		return
	}
	if len(args) == 0 {
		stdlog.Print(l.prefix + f + "\n")
	} else {
		stdlog.Printf(l.prefix+f+"\n", args...)
	}
}

func (l logAPI) Fatal(f string, args ...interface{}) {
	if len(args) == 0 {
		stdlog.Fatalf(l.prefix + f + "\n")
	} else {
		stdlog.Fatalf(l.prefix+f+"\n", args...)
	}
}

func (l logAPI) Error(f string, args ...interface{}) {
	if len(args) == 0 {
		stdlog.Print(l.prefix + f + "\n")
	} else {
		stdlog.Printf(l.prefix+f+"\n", args...)
	}
}
