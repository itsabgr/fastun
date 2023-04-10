package fastun

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

var client = http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}}

func getFallBack(ctx context.Context, fallback string) (*http.Response, error) {
	if fallback == "" {
		return nil, errors.New("no fallback")
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fallback, nil)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}
func Serve(addr string, cors string, fallback string, debug bool) error {
	if cors == "" {
		cors = "*"
	}
	return (&fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			switch string(ctx.Method()) {
			case fasthttp.MethodGet, fasthttp.MethodHead, fasthttp.MethodOptions:
				ctx.Response.Header.Add("Access-Control-Allow-Origin", cors)
				ctx.Response.Header.Add("Access-Control-Allow-Methods", "GET, OPTIONS, PUT")
				ctx.Response.Header.Add("Access-Control-Allow-Headers", "*")
				ctx.Response.Header.SetConnectionClose()
				resp, err := getFallBack(ctx, fallback)
				if err != nil {
					if debug {
						ctx.SetBodyString(fmt.Sprintf("%d\n%s\n%s", time.Now().UTC().Unix(), ctx.RemoteIP().String(), err.Error()))
					} else {
						ctx.SetBodyString(fmt.Sprintf("%d\n%s", time.Now().UTC().Unix(), ctx.RemoteIP().String()))
					}
					return
				}
				ctx.SetStatusCode(resp.StatusCode)
				if loc := resp.Header.Get("Location"); loc != "" {
					ctx.Response.Header.Add("Location", loc)
				}
				ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
					defer resp.Body.Close()
					io.Copy(w, resp.Body)
				})
				return
			}
			target := strings.Trim(string(ctx.Path()), "/")
			conn, err := net.DialTimeout("tcp", target, 1*time.Second)
			if err != nil {
				ctx.SetStatusCode(400)
				if debug {
					ctx.SetBodyString(err.Error())
				}
				return
			}
			if ctx.IsBodyStream() {
				go io.Copy(conn, ctx.RequestBodyStream())
			} else {
				go conn.Write(ctx.PostBody())
			}
			ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
				defer conn.Close()
				io.Copy(w, conn)
			})
		},
		LogAllErrors:                  false,
		Logger:                        log.New(io.Discard, "", 0),
		StreamRequestBody:             true,
		MaxRequestBodySize:            1,
		DisableHeaderNamesNormalizing: true,
		DisablePreParseMultipartForm:  true,
		WriteTimeout:                  time.Second * 30,
		ReadTimeout:                   time.Second * 30,
		SecureErrorLogMessage:         true,
		ErrorHandler: func(ctx *fasthttp.RequestCtx, err error) {
			ctx.SetStatusCode(500)
		},
		MaxRequestsPerConn:                 1,
		CloseOnShutdown:                    true,
		SleepWhenConcurrencyLimitsExceeded: time.Second,
		TCPKeepalive:                       true,
		TCPKeepalivePeriod:                 time.Second * 3,
		MaxIdleWorkerDuration:              time.Second * 5,
		IdleTimeout:                        time.Second * 2,
		NoDefaultDate:                      true,
		DisableKeepalive:                   true,
		NoDefaultServerHeader:              true,
		NoDefaultContentType:               true,
	}).ListenAndServe(addr)
}
