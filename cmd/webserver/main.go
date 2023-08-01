package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"runtime/debug"
	"sync"
	"sync/atomic"

	env "github.com/Netflix/go-env"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"golang.org/x/exp/slog"
)

const MaxUint16 = uint32(^uint16(0))

type ConnectionSubscriptions struct {
	sync.Mutex                     // todo profile and compare with sync.Map
	subs         map[uint32]string // todo profile and compare with sync.Map
	topics       map[string]uint16
	topicCounter uint16
}

func NewConnectionSubscriptions() *ConnectionSubscriptions {
	cs := &ConnectionSubscriptions{}
	cs.subs = make(map[uint32]string, 32)
	cs.topics = make(map[string]uint16, 32)
	cs.topicCounter = 0

	return cs
}
func (cs *ConnectionSubscriptions) Subscribe(connectionID uint32, topic string) {
	cs.Lock()
	defer cs.Unlock()
	cs.subs[connectionID] = topic
}
func (cs *ConnectionSubscriptions) Unsubscribe(connectionID uint32, topic string) {
	cs.Lock()
	defer cs.Unlock()
	delete(cs.subs, connectionID)
}
func (cs *ConnectionSubscriptions) GetTopic(connectionID uint32) string {
	cs.Lock()
	defer cs.Unlock()
	return cs.subs[connectionID]
}

type ConnectionCommunicationHolder struct {
	sync.Mutex                        // todo profile and compare with sync.Map
	allChans   map[uint32]chan uint64 // todo profile and compare with sync.Map
}

func NewConnectionCommunicationHolder() *ConnectionCommunicationHolder {
	allCh := make(map[uint32]chan uint64, 32)
	return &ConnectionCommunicationHolder{
		allChans: allCh,
	}
}
func (cch *ConnectionCommunicationHolder) GetChan(connectionID uint32) (ch chan uint64, ok bool) {
	cch.Lock()
	defer cch.Unlock()

	ch, ok = cch.allChans[connectionID]
	return ch, ok
}
func (cch *ConnectionCommunicationHolder) SetChan(connectionID uint32, ch chan uint64) {
	cch.Lock()
	defer cch.Unlock()

	cch.allChans[connectionID] = ch
}
func (cch *ConnectionCommunicationHolder) DeleteAndCloseChan(connectionID uint32) {
	cch.Lock()
	defer cch.Unlock()

	ch, ok := cch.allChans[connectionID]
	if ok {
		delete(cch.allChans, connectionID)
		close(ch)
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	opts := slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	handler := slog.NewJSONHandler(os.Stdout, &opts)
	buildInfo, _ := debug.ReadBuildInfo()
	logger := slog.New(handler).With(
		slog.Group("program_info",
			slog.Int("pid", os.Getpid()),
			slog.String("go_version", buildInfo.GoVersion),
		),
	)
	var config Environment
	_, err := env.UnmarshalFromEnviron(&config)
	if err != nil {
		logger.Error("error reading env", slog.Any("err", err))
		panic(err)
	}

	logger.Info("starting...")

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		startWebServer(ctx, config.WebsocketPort, logger)
	}()

	logger.Debug("after goroutine started...")

	wg.Wait()
	logger.Info("signal received, done")
}

func startWebServer(ctx context.Context, wsPort int, logger *slog.Logger) {
	gSubs := NewConnectionSubscriptions()
	commonBus := make(chan uint64)
	allChans := NewConnectionCommunicationHolder()
	var globalCounter atomic.Uint32
	globalCounter.Store(0)
	rxp, _ := regexp.Compile("^/?ws/canvas/([a-zA-Z0-9]{4,20})/?$")
	go func() {
		logger.Debug("loop commonBus")
		for v := range commonBus {
			for i := uint32(0); i <= globalCounter.Load(); i++ {
				if gSubs.GetTopic(i) == "" {
					continue
				}
				ch, ok := allChans.GetChan(i)
				if ok && ch != nil {
					ch <- v
				}
			}
		}
		logger.Debug("[END] loop commonBus")
	}()
	s := http.Server{Addr: fmt.Sprintf(":%d", wsPort), Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Debug("[host]", slog.StringValue(r.Host))
		logger.Debug("[headers]", slog.AnyValue(r.Header))
		regexpResult := rxp.FindAllString(r.URL.Path, 1)
		if len(regexpResult) != 1 {
			logger.Debug("regexp failed", slog.String("path", r.URL.Path))
			w.WriteHeader(http.StatusNotFound)
			return
		}
		topic := regexpResult[0]
		conn, _, wh, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			logger.Error("[serving] err1", slog.Any("err", err))
			// handle error
			return
		}
		logger.Debug("[wh proto]", slog.StringValue(wh.Protocol))
		gi := globalCounter.Add(1)
		gi -= 1
		if gi > MaxUint16 {
			logger.Error("[error] max connections exceeded", slog.Any("err", err))
			conn.Close()
			return
		}
		myChan := make(chan uint64)
		allChans.SetChan(gi, myChan)
		gSubs.Subscribe(gi, topic)
		closeConn := func() {
			conn.Close()
			allChans.DeleteAndCloseChan(gi)
			gSubs.Unsubscribe(gi, topic)
			logger.Debug("closed chan gi", slog.Uint64("gi", uint64(gi)))
		}
		go func() {
			defer closeConn()
			logger.Debug("[serving] listening")
			i := 0
			kb := 0
			for {
				msg, op, err := wsutil.ReadClientData(conn)
				if err != nil {
					logger.Error("[serving] err2", slog.String("msg", string(msg)), slog.Any("op", op), slog.Any("err", err))
					// handle error
					return
				}

				if op != ws.OpBinary {
					err = wsutil.WriteServerMessage(conn, op, msg)
					if err != nil {
						logger.Debug("[serving] err3", slog.Any("err", err))
						return
					}
					logger.Debug("respond right away", slog.Any("op", op))
					continue
				}
				kb += len(msg)
				i++

				if i%100 == 0 {
					logger.Debug("[serving] listening, got messages", slog.Int("i", i), slog.Int("kb", kb/1024), slog.Any("msg", msg))
				}

				receivedCoords := binary.BigEndian.Uint32(msg)
				signedCoords := make([]byte, 0, 8)
				signedCoords = binary.LittleEndian.AppendUint16(signedCoords, uint16(gi))
				signedCoords = binary.BigEndian.AppendUint32(signedCoords, receivedCoords)
				signedCoords = binary.BigEndian.AppendUint16(signedCoords, uint16(0))

				commonBus <- binary.BigEndian.Uint64(signedCoords)
			}
		}()
		go func() {
			defer closeConn()
			logger.Debug("[serving] waiting for write", slog.Uint64("gi", uint64(gi)))
			for newV := range myChan {
				excessive64Buff := make([]byte, 0, 8)
				excessive64Buff = binary.BigEndian.AppendUint64(excessive64Buff, newV)
				buff := make([]byte, 0, 6)
				buff = append(buff, excessive64Buff[:6]...)

				err = wsutil.WriteServerMessage(conn, ws.OpBinary, buff)
				if err != nil {
					logger.Debug("[serving] err4", slog.Any("err", err))
					return
				}
			}
		}()
	})}
	go func() {
		logger.Debug("ListenAndServe", slog.Int("port", wsPort))
		err := s.ListenAndServe()
		logger.Debug("after ListenAndServe", slog.Any("err", err))
	}()

	<-ctx.Done()
	logger.Info("shutting down a server")
	s.Shutdown(ctx)
}
