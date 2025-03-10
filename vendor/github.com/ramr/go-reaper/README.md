
# go-reaper

Process (grim) reaper library for golang - this is useful for cleaning up
zombie processes inside docker containers (which do not have an init
process running as pid 1).

## tl;dr

```go
import reaper "github.com/ramr/go-reaper"

func main() {
        //  Start background reaping of orphaned child processes.
        go reaper.Reap()

        //  Rest of your code ...

        //  Note: If you also manage processes within your code aka
        //        exec commands or include some code that does do that,
        //        please refer to the section titled
        //        "[Into The Woods]"(https://github.com/ramr/go-reaper#into-the-woods)
}

```

## How and Why

If you run a container without an init process (pid 1) which would
normally reap zombie processes, you could well end up with a lot of zombie
processes and eventually exhaust the max process limit on your system.

If you have a golang program that runs as pid 1, then this library allows
the golang program to setup a background signal handling mechanism to
handle the death of those orphaned children and not create a load of
zombies inside the pid namespace your container runs in.

## Usage

For basic usage, see the tl;dr section above. This should be the
most commonly used route you'd need to take.

## Road Less Traveled

But for those that prefer to go down "the road less traveled", you can
control whether to disable pid 1 checks and/or control the options passed
to the `wait4` (or `waitpid`) system call by passing configuration to the
reaper, enable subreaper functionality and optionally get notified when
child processes are reaped.

```go
import reaper "github.com/ramr/go-reaper"

func main() {
        config := reaper.Config{
                Pid:                  0,
                Options:              0,
                Debug:                true,
                DisablePid1Check:     false,
                EnableChildSubreaper: false,

                //  If you wish to get notified whenever a child process is
                //  reaped, use a `buffered` status channel.
                //      StatusChannel: make(chan reaper.Status, 42),
                StatusChannel: nil,
        }

        //  Only use this if you care about status notifications
        //  for reaped process (aka StatusChannel != nil).
        if config.StatusChannel != nil {
                go func() {
                        select {
                        case status, ok := <-config.StatusChannel:
                                if !ok {
                                        return
                                }
                                // process status (reaper.Status)
                        }
                }()
        }

        //  Start background reaping of orphaned child processes.
        go reaper.Start(config)

        //  Rest of your code ...
}

```

The `Pid` and `Options` fields in the configuration are the `pid` and
`options` passed to the linux `wait4` system call.

See the man pages for the [wait4](https://linux.die.net/man/2/wait4) or
[waitpid](https://linux.die.net/man/2/waitpid) system call for details.

## Into The Woods

And finally, this part is for those folks that want to go into the woods.
This could be required when you need to manage the processes you invoke
inside your code (ala with `os.exec.Command*` or `syscall.ForkExec` or any
variants) or basically include some libraries/code that need to do the same.
In such a case, it is better to run the reaper in a separate process
as `pid 1` and run your code inside a child process. This will still be
part of the same code base but just `forked` off so that the reaper runs
inside a different process ...

```go
import (
        "os"
        "syscall"

        reaper "github.com/ramr/go-reaper"
)

func main() {
        // Use an environment variable REAPER to indicate whether or not
        // we are the child/parent.
        if _, hasReaper := os.LookupEnv("REAPER"); !hasReaper {
                //  Start background reaping of orphaned child processes.
                go reaper.Reap()

                // Note: Optionally add an argument to the end to more
                //       easily distinguish the parent and child in
                //       something like `ps` etc.
                args := os.Args
                // args := append(os.Args, "#kiddo")

                pwd, err := os.Getwd()
                if err != nil {
                        // Note: Better to use a default dir ala "/tmp".
                        panic(err)
                }

                kidEnv := []string{ fmt.Sprintf("REAPER=%d", os.Getpid()) }

                var wstatus syscall.WaitStatus
                pattrs := &syscall.ProcAttr{
                        Dir:   pwd,
                        Env:   append(os.Environ(), kidEnv...),
                        Sys:   &syscall.SysProcAttr{Setsid: true},
                        Files: []uintptr{
                                uintptr(syscall.Stdin),
                                uintptr(syscall.Stdout),
                                uintptr(syscall.Stderr),
                        },
                }

                pid, _ := syscall.ForkExec(args[0], args, pattrs)

                // fmt.Printf("kiddo-pid = %d\n", pid)
                _, err = syscall.Wait4(pid, &wstatus, 0, nil)
                for syscall.EINTR == err {
                        _, err = syscall.Wait4(pid, &wstatus, 0, nil)
                }

                // If you put this code into a function, then exit here.
                os.Exit(0)
                return
        }

        //  Rest of your code goes here ...

}  /*  End of func  main.  */

```
