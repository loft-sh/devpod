//go:build profile

package cmd

import (
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
)

func init() {
	go func() {
		myMux := http.NewServeMux()

		myMux.HandleFunc("/debug/pprof/", pprof.Index)
		myMux.HandleFunc("/debug/pprof/{action}", pprof.Index)
		myMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			return
		}

		f, err := os.OpenFile("/tmp/pprof_ports", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			f.Write([]byte(fmt.Sprintf("%d=%d\n", os.Getpid(), listener.Addr().(*net.TCPAddr).Port)))
			f.Close()
		}

		http.Serve(listener, myMux)
	}()
}
