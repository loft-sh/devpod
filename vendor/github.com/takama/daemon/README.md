# Go Daemon

A daemon package for use with Go (golang) services

[![GoDoc](https://godoc.org/github.com/takama/daemon?status.svg)](https://godoc.org/github.com/takama/daemon)

## Examples

### Simplest example (just install self as daemon)

```go
package main

import (
    "fmt"
    "log"

    "github.com/takama/daemon"
)

func main() {
    service, err := daemon.New("name", "description", daemon.SystemDaemon)
    if err != nil {
        log.Fatal("Error: ", err)
    }
    status, err := service.Install()
    if err != nil {
        log.Fatal(status, "\nError: ", err)
    }
    fmt.Println(status)
}
```

### Real example

```go
// Example of a daemon with echo service
package main

import (
    "fmt"
    "log"
    "net"
    "os"
    "os/signal"
    "syscall"

    "github.com/takama/daemon"
)

const (

    // name of the service
    name        = "myservice"
    description = "My Echo Service"

    // port which daemon should be listen
    port = ":9977"
)

//    dependencies that are NOT required by the service, but might be used
var dependencies = []string{"dummy.service"}

var stdlog, errlog *log.Logger

// Service has embedded daemon
type Service struct {
    daemon.Daemon
}

// Manage by daemon commands or run the daemon
func (service *Service) Manage() (string, error) {

    usage := "Usage: myservice install | remove | start | stop | status"

    // if received any kind of command, do it
    if len(os.Args) > 1 {
        command := os.Args[1]
        switch command {
        case "install":
            return service.Install()
        case "remove":
            return service.Remove()
        case "start":
            return service.Start()
        case "stop":
            return service.Stop()
        case "status":
            return service.Status()
        default:
            return usage, nil
        }
    }

    // Do something, call your goroutines, etc

    // Set up channel on which to send signal notifications.
    // We must use a buffered channel or risk missing the signal
    // if we're not ready to receive when the signal is sent.
    interrupt := make(chan os.Signal, 1)
    signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)

    // Set up listener for defined host and port
    listener, err := net.Listen("tcp", port)
    if err != nil {
        return "Possibly was a problem with the port binding", err
    }

    // set up channel on which to send accepted connections
    listen := make(chan net.Conn, 100)
    go acceptConnection(listener, listen)

    // loop work cycle with accept connections or interrupt
    // by system signal
    for {
        select {
        case conn := <-listen:
            go handleClient(conn)
        case killSignal := <-interrupt:
            stdlog.Println("Got signal:", killSignal)
            stdlog.Println("Stoping listening on ", listener.Addr())
            listener.Close()
            if killSignal == os.Interrupt {
                return "Daemon was interruped by system signal", nil
            }
            return "Daemon was killed", nil
        }
    }

    // never happen, but need to complete code
    return usage, nil
}

// Accept a client connection and collect it in a channel
func acceptConnection(listener net.Listener, listen chan<- net.Conn) {
    for {
        conn, err := listener.Accept()
        if err != nil {
            continue
        }
        listen <- conn
    }
}

func handleClient(client net.Conn) {
    for {
        buf := make([]byte, 4096)
        numbytes, err := client.Read(buf)
        if numbytes == 0 || err != nil {
            return
        }
        client.Write(buf[:numbytes])
    }
}

func init() {
    stdlog = log.New(os.Stdout, "", log.Ldate|log.Ltime)
    errlog = log.New(os.Stderr, "", log.Ldate|log.Ltime)
}

func main() {
    srv, err := daemon.New(name, description, daemon.SystemDaemon, dependencies...)
    if err != nil {
        errlog.Println("Error: ", err)
        os.Exit(1)
    }
    service := &Service{srv}
    status, err := service.Manage()
    if err != nil {
        errlog.Println(status, "\nError: ", err)
        os.Exit(1)
    }
    fmt.Println(status)
}
```

### Service config file

Optionally, service config file can be retrieved or updated by calling
`GetTemplate() string` and `SetTemplate(string)` methods(except MS
Windows). Template will be a default Go Template(`"text/template"`).

If `SetTemplate` is not called, default template content will be used
while creating service.

| Variable     | Description                      |
| ------------ | -------------------------------- |
| Description  | Description for service          |
| Dependencies | Service dependencies             |
| Name         | Service name                     |
| Path         | Path of service executable       |
| Args         | Arguments for service executable |

#### Example template(for linux systemv)

```ini
[Unit]
Description={{.Description}}
Requires={{.Dependencies}}
After={{.Dependencies}}

[Service]
PIDFile=/var/run/{{.Name}}.pid
ExecStartPre=/bin/rm -f /var/run/{{.Name}}.pid
ExecStart={{.Path}} {{.Args}}
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

### Cron example

See `examples/cron/cron_job.go`

## Contributors (unsorted)

- [Sheile](https://github.com/Sheile)
- [Nguyen Trung Loi](https://github.com/loint)
- [Donny Prasetyobudi](https://github.com/donnpebe)
- [Mark Berner](https://github.com/mark2b)
- [Fatih Kaya](https://github.com/fatihky)
- [Jannick Fahlbusch](https://github.com/jannickfahlbusch)
- [TobyZXJ](https://github.com/tobyzxj)
- [Pichu Chen](https://github.com/PichuChen)
- [Eric Halpern](https://github.com/ehalpern)
- [Yota](https://github.com/nus)
- [Erkan Durmus](https://github.com/derkan)
- [maxxant](https://github.com/maxxant)
- [1for](https://github.com/1for)
- [okamura](https://github.com/sidepelican)
- [0X8C - Demired](https://github.com/Demired)
- [Maximus](https://github.com/maximus12793)
- [AlgorathDev](https://github.com/AlgorathDev)
- [Alexis Camilleri](https://github.com/krysennn)
- [neverland4u](https://github.com/neverland4u)
- [Rustam](https://github.com/rusq)
- [King'ori Maina](https://github.com/itskingori)

All the contributors are welcome. If you would like to be the contributor please accept some rules.

- The pull requests will be accepted only in `develop` branch
- All modifications or additions should be tested
- Sorry, We will not accept code with any dependency, only standard library

Thank you for your understanding!

## License

[MIT Public License](https://github.com/takama/daemon/blob/master/LICENSE)
