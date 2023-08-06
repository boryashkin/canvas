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

	env "github.com/Netflix/go-env"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"golang.org/x/exp/slog"

	"github.com/boryashkin/canvas/internal/pubsub"
)

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
	gSubs := pubsub.NewConnectionSubscriptionsMap()
	rxp, _ := regexp.Compile("^/?ws/canvas/([a-zA-Z0-9]{4,20})/?$")

	go func() {
		logger.Debug("loop commonBus")
		gSubs.RunNotificationLoop(ctx)
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
		gi, myChan, err := gSubs.GetNewConnectionIdAndChan()
		if err != nil {
			logger.Error("failed to handle a connection", slog.Any("err", err))
			conn.Close()
			return
		}
		gSubs.Subscribe(gi, topic)
		closeConn := func() {
			conn.Close()
			gSubs.DeleteConnection(gi)
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

				gSubs.Publish(topic, binary.BigEndian.Uint64(signedCoords))
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
