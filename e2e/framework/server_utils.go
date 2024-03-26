package framework

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// serveAgent will be a simple http file server that will expose our
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

	randomPort, err := findOpenPort(0)
	if err != nil {
		log.Fatal(err)
	}

	ip := getIP()
	addr := fmt.Sprintf("%s:%d", ip, randomPort)

	err = os.Setenv("DEVPOD_AGENT_URL", "http://"+addr+"/files/")
	if err != nil {
		log.Fatal(err)
	}

	// Start the HTTP server on port 8080
	log.Printf("Server started on %s", addr)
	err = http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func findOpenPort(retries int) (int, error) {
	if retries > 100 {
		return 0, fmt.Errorf("no open port available in the range")
	}
	// Create a new random number generator with a custom seed (e.g., current time)
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	min := 10000
	max := 40000
	port := rng.Intn(max-min+1) + min

	conn, err := net.Dial("tcp", net.JoinHostPort("localhost", strconv.Itoa(port)))
	if err == nil {
		conn.Close()
		return findOpenPort(retries + 1)
	}

	return port, nil
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
