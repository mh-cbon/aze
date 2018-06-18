package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	stdlog "log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"time"

	"code.cloudfoundry.org/bytefmt"
	"github.com/juju/ratelimit"
)

var (
	log = logAPI{}
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var opts = struct {
		quiet   bool
		timeout time.Duration
	}{}

	flag.DurationVar(&opts.timeout, "t", time.Second*60, "timeout of the conenction")
	flag.BoolVar(&opts.quiet, "q", false, "stfu")

	flag.Parse()

	log.SetQuiet(opts.quiet)

	// parse <dst> <src> <cap>
	args := flag.Args()

	{
		if len(args) == 3 && args[0] == "gen" {
			var cnt uint64
			{
				c, err := bytefmt.ToBytes(args[1])
				if err != nil {
					log.Fatal("failed to parse %q, err=%v", args[1], err)
				}
				cnt = c
			}
			sampleStr := args[2]
			sampleLen := uint64(len(sampleStr))
			sample := bytes.Repeat([]byte(sampleStr), len(sampleStr))
			var i uint64
			for i = 0; i < cnt; i += sampleLen {
				os.Stdout.Write(sample)
			}
			log.Print("copied %v", bytefmt.ByteSize(i))
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
			log.Fatal("faield to parse capacity %q, err=%v", cap, err)
		}
		capBytes = b
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
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			var src net.Conn
			{
				c, err := l.Accept()
				if err != nil {
					log.Error("accept error %v", err)
					continue
				}
				defer c.Close()
				src = &idleTimeoutConn{Conn: c, timeout: opts.timeout}
			}
			go func() {
				var dst net.Conn
				{
					d, err := net.Dial(dstProto, dstAddr)
					if err != nil {
						log.Error("dial error %v", err)
						return
					}
					defer d.Close()
					dst = &idleTimeoutConn{Conn: d, timeout: opts.timeout}
				}
				if err := copy(src, dst, capBytes); err != nil {
					log.Error("failed to handle %v", err)
				}
			}()
		}
	}()

	<-cancelNotifier()
	cancel()

}

func copy(src, dst net.Conn, capBytes uint64) error {
	bucket := ratelimit.NewBucketWithRate(float64(capBytes), int64(capBytes))
	n, err := io.Copy(ratelimit.Writer(dst, bucket), src)
	// n, err := io.Copy(dst, src)
	if err != nil {
		return fmt.Errorf("copy error %v", err)
	}
	log.Print("%v -> %v copied %v bytes", src.RemoteAddr(), dst.RemoteAddr(), n)
	return nil
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
	stfu bool
}

func (l *logAPI) SetQuiet(stfu bool) {
	l.stfu = stfu
}

func (l logAPI) Print(f string, args ...interface{}) {
	if l.stfu {
		return
	}
	if len(args) == 0 {
		stdlog.Print(f + "\n")
	} else {
		stdlog.Printf(f+"\n", args...)
	}
}

func (l logAPI) Fatal(f string, args ...interface{}) {
	if len(args) == 0 {
		stdlog.Fatalf(f + "\n")
	} else {
		stdlog.Fatalf(f+"\n", args...)
	}
}

func (l logAPI) Error(f string, args ...interface{}) {
	if len(args) == 0 {
		stdlog.Print(f + "\n")
	} else {
		stdlog.Printf(f+"\n", args...)
	}
}
