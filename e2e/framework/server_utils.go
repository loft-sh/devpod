package framework

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
)

// ServeAgent will be a simple http file server that will expose our
// freshly compiled devpod binaries to be downloaded as agents.
// useful for non-linux runners
func ServeAgent() {
	// Specify the directory containing the files you want to serve
	dir := "bin"

	wd, err := os.Getwd()
	if err == nil {
		dir = filepath.Join(wd, "bin")
	}

	// Create a file server handler for the specified directory
	fileServer := http.FileServer(http.Dir(dir))

	// Register the file server handler to serve files under the /files route
	http.Handle("/files/", http.StripPrefix("/files", fileServer))

	ip := getIP()

	listener, err := net.Listen("tcp", fmt.Sprintf("%v:0", ip))
	if err != nil {
		log.Fatal(err)
	}

	addr := listener.Addr().String()
	err = os.Setenv("DEVPOD_AGENT_URL", "http://"+addr+"/files/")
	if err != nil {
		log.Fatal(err)
	}

	// Start the HTTP server on port 8080
	log.Printf("Server started on %s", addr)

	err = http.Serve(listener, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func getIP() string {
	// Get a list of network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		return "0.0.0.0"
	}

	// Iterate over each network interface
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return "0.0.0.0"
		}

		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPAddr:
				if v.IP.To4() != nil {
					if v.IP.DefaultMask().String() == "ffffff00" || v.IP.DefaultMask().String() == "ff000000" {
						return v.IP.String()
					}
				}
			case *net.IPNet:
				if v.IP.To4() != nil {
					if v.IP.DefaultMask().String() == "ffffff00" || v.IP.DefaultMask().String() == "ff000000" {
						return v.IP.String()
					}
				}
			}
		}
	}

	return "0.0.0.0"
}
