package main

import ( "fmt" "time"

"github.com/go-vgo/robotgo"
"github.com/micmonay/keybd_event"

)

var lastActivityTime time.Time

func detectKeyboardActivity() { kb, err := keybd_event.NewKeyBonding() if err != nil { fmt.Println("Error setting up keyboard listener:", err) return }

for {
	time.Sleep(1 * time.Second) // Check every second
	if kb.HasCTRL() || kb.HasALT() || kb.HasSHIFT() {
		lastActivityTime = time.Now()
	}
}

}

func detectMouseActivity() { lastX, lastY := robotgo.GetMousePos() for { time.Sleep(1 * time.Second) // Check every second x, y := robotgo.GetMousePos() if x != lastX || y != lastY { lastActivityTime = time.Now() lastX, lastY = x, y } } }

func inactivityWatcher(timeout time.Duration) { for { time.Sleep(1 * time.Second) if time.Since(lastActivityTime) > timeout { fmt.Println("User is inactive!") break } } }

func main() { lastActivityTime = time.Now()

go detectKeyboardActivity()
go detectMouseActivity()
go inactivityWatcher(10 * time.Second) // Set timeout to 10s

select {} // Keep running

}
