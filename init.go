// grace provides for graceful restart for go http servers.
// There are 2 parts to graceful restarts
// 1. Share listening sockets (this is done via socketmaster binary)
// 2. Close listener gracefully (via graceful)
package grace

import (
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"time"

	graceful "gopkg.in/tylerb/graceful.v1"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/reuseport"
)

var listenPort string
var cfgtestFlag bool

// Config grace package config
type Config struct {
	Timeout          time.Duration
	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration
}

// add -p flag to the list of flags supported by the app,
// and allow it to over-ride default listener port in config/app
func init() {
	flag.StringVar(&listenPort, "p", "", "listener port")
	flag.BoolVar(&cfgtestFlag, "t", false, "config test")
}

// GetListenPort applications need some way to access the port
// TODO: this method will work only after grace.Serve is called.
func GetListenPort(hport string) string {
	return listenPort
}

// ServerFastHTTP use fasthttp server
func ServerFastHTTP(hport string, handler fasthttp.RequestHandler) error {
	var l net.Listener
	var err error

	fd := os.Getenv("EINHORN_FDS")
	if fd != "" {
		sock, err := strconv.Atoi(fd)
		if err == nil {
			hport = "socketmaster:" + fd
			log.Println("detected socketmaster, listening on", fd)
			file := os.NewFile(uintptr(sock), "listener")

			if err := syscall.SetsockoptInt(sock, syscall.SOL_SOCKET, 0x0F, 1); err != nil {
				return err
			}
			fl, err := net.FileListener(file)
			if err == nil {
				l = fl
			}
		}
	}

	if listenPort != "" {
		hport = ":" + listenPort
	}

	if l == nil {
		l, err = reuseport.Listen("tcp4", hport)
		if err != nil {
			log.Fatalf("error in reuseport listener: %s", err)
		}

	}

	server := fasthttp.Server{
		ReadBufferSize: 4096 * 4,
		Handler:        handler,
	}

	log.Println("starting serve on ", hport)
	return server.Serve(l)

}

// Serve start serving on hport. If running via socketmaster, the hport argument is
// ignored. Also, if a port was specified via -p, it takes precedence on hport
func Serve(hport string, handler http.Handler) error {
	config := Config{
		Timeout:          10 * time.Second,
		HTTPReadTimeout:  5 * time.Second,
		HTTPWriteTimeout: 10 * time.Second,
	}

	return ServeWithConfig(hport, config, handler)
}

// ServeWithConfig serve using package config
func ServeWithConfig(hport string, config Config, handler http.Handler) error {
	checkConfigTest()

	l, err := Listen(hport)
	if err != nil {
		log.Fatalln(err)
	}

	srv := &graceful.Server{
		Timeout: config.Timeout,
		Server: &http.Server{
			Handler:      handler,
			ReadTimeout:  config.HTTPReadTimeout,
			WriteTimeout: config.HTTPWriteTimeout,
		},
	}

	log.Println("starting serve on ", hport)
	return srv.Serve(l)
}

// Run exports Run() from grace package
func Run(addr string, timeout time.Duration, n http.Handler) {
	graceful.Run(addr, timeout, n)
}

// RunWithErr exports RunWithErr from grace package
func RunWithErr(addr string, timeout time.Duration, n http.Handler) error {
	return graceful.RunWithErr(addr, timeout, n)
}

// Listen This method can be used for any TCP Listener, e.g. non HTTP
func Listen(hport string) (net.Listener, error) {
	var l net.Listener

	fd := os.Getenv("EINHORN_FDS")
	if fd != "" {
		sock, err := strconv.Atoi(fd)
		if err == nil {
			hport = "socketmaster:" + fd
			log.Println("detected socketmaster, listening on", fd)
			file := os.NewFile(uintptr(sock), "listener")
			fl, err := net.FileListener(file)
			if err == nil {
				l = fl
			}
		}
	}

	if listenPort != "" {
		hport = ":" + listenPort
	}

	checkConfigTest()

	if l == nil {
		var err error
		l, err = net.Listen("tcp4", hport)
		if err != nil {
			return nil, err
		}
	}

	return l, nil
}

func checkConfigTest() {
	if cfgtestFlag == true {
		log.Println("config test mode, exiting")
		os.Exit(0)
	}
}
