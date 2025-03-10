package reaper

/*  Note:  This is a *nix only implementation.  */

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

// Reaper configuration.
type Config struct {
	Pid                  int
	Options              int
	DisablePid1Check     bool
	EnableChildSubreaper bool
	StatusChannel        chan Status
	Debug                bool
}

// Reaped child process status information.
type Status struct {
	Pid        int
	Err        error
	WaitStatus syscall.WaitStatus
}

// Send the child status on the status `ch` channel.
func notify(ch chan Status, pid int, err error, ws syscall.WaitStatus) {
	if ch == nil {
		return
	}

	status := Status{Pid: pid, Err: err, WaitStatus: ws}

	// The only case for recovery would be if the caller closes the
	// `StatusChannel`. That is not really something recommended or
	// as the normal `contract` is that the writer would close the
	// channel as an EOF/EOD indicator.
	// But stranger things have (sic) actually happened ...
	defer func() {
		r := recover()
		if r == nil {
			return
		}

		fmt.Printf(" - Recovering from notify panic: %v\n", r)
		fmt.Printf(" - Lost pid %v status: %+v\n", pid, status)
	}()

	select {
	case ch <- status: /*  Notified with the child status.  */
	default: /*  blocked ... channel full or no reader!  */
		fmt.Printf(" - Status channel full, lost pid %v: %+v\n",
			pid, status)
	}

} /*  End of function  notify.  */

// Handle death of child messages (SIGCHLD). Pushes the signal onto the
// notifications channel if there is a waiter.
func sigChildHandler(notifications chan os.Signal) {
	var sigs = make(chan os.Signal, 3)
	signal.Notify(sigs, syscall.SIGCHLD)

	for {
		var sig = <-sigs
		select {
		case notifications <- sig: /*  published it.  */
		default:
			/*
			 *  Notifications channel full - drop it to the
			 *  floor. This ensures we don't fill up the SIGCHLD
			 *  queue. The reaper just waits for any child
			 *  process (pid=-1), so we ain't loosing it!! ;^)
			 */
		}
	}

} /*  End of function  sigChildHandler.  */

// Be a good parent - clean up behind the children.
func reapChildren(config Config) {
	var notifications = make(chan os.Signal, 1)

	go sigChildHandler(notifications)

	pid := config.Pid
	opts := config.Options
	informer := config.StatusChannel

	for {
		var sig = <-notifications
		if config.Debug {
			fmt.Printf(" - Received signal %v\n", sig)
		}
		for {
			var wstatus syscall.WaitStatus

			/*
			 *  Reap 'em, so that zombies don't accumulate.
			 *  Plants vs. Zombies!!
			 */
			pid, err := syscall.Wait4(pid, &wstatus, opts, nil)
			for syscall.EINTR == err {
				pid, err = syscall.Wait4(pid, &wstatus, opts, nil)
			}

			if syscall.ECHILD == err {
				break
			}

			if config.Debug {
				fmt.Printf(" - Grim reaper cleanup: pid=%d, wstatus=%+v\n",
					pid, wstatus)
			}

			if informer != nil {
				go notify(informer, pid, err, wstatus)
			}
		}
	}

} /*   End of function  reapChildren.  */

/*
 *  ======================================================================
 *  Section: Exported functions
 *  ======================================================================
 */

// Normal entry point for the reaper code. Start reaping children in the
// background inside a goroutine.
func Reap() {
	/*
	 *  Only reap processes if we are taking over init's duties aka
	 *  we are running as pid 1 inside a docker container. The default
	 *  is to reap all processes.
	 */
	Start(Config{
		Pid:                  -1,
		Options:              0,
		DisablePid1Check:     false,
		EnableChildSubreaper: false,
	})

} /*  End of [exported] function  Reap.  */

// Entry point for invoking the reaper code with a specific configuration.
// The config allows you to bypass the pid 1 checks, so handle with care.
// The child processes are reaped in the background inside a goroutine.
func Start(config Config) {
	/*
	 *  Start the Reaper with configuration options. This allows you to
	 *  reap processes even if the current pid isn't running as pid 1.
	 *  So ... use with caution!!
	 *
	 *  In most cases, you are better off just using Reap() as that
	 *  checks if we are running as Pid 1.
	 */
	if config.EnableChildSubreaper {
		/*
		 *  Enabling the child sub reaper means that any orphaned
		 *  descendant process will get "reparented" to us.
		 *  And we then do the reaping when those processes die.
		 */
		fmt.Println(" - Enabling child subreaper ...")
		err := unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0)
		if err != nil {
			// Log the error and continue ...
			fmt.Printf(" - Error enabling subreaper: %v\n", err)
		}
	}

	if !config.DisablePid1Check {
		mypid := os.Getpid()
		if 1 != mypid {
			fmt.Println(" - Grim reaper disabled, pid not 1")
			return
		}
	}

	/*
	 *  Ok, so either pid 1 checks are disabled or we are the grandma
	 *  of 'em all, either way we get to play the grim reaper.
	 *  You will be missed, Terry Pratchett!! RIP
	 */
	go reapChildren(config)

} /*  End of [exported] function  Start.  */
