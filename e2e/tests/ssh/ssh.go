package ssh

import (
	"bufio"
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/loft-sh/devpod/e2e/framework"
	"github.com/onsi/ginkgo/v2"
)

var _ = DevPodDescribe("devpod ssh test suite", func() {
	ginkgo.Context("testing ssh command", ginkgo.Label("ssh"), ginkgo.Ordered, func() {
		ctx := context.Background()
		initialDir, err := os.Getwd()
		if err != nil {
			panic(err)
		}

		ginkgo.It("should start a new workspace with a docker provider (default) and ssh into it", func() {
			tempDir, err := framework.CopyToTempDir("tests/ssh/testdata/local-test")
			framework.ExpectNoError(err)
			defer framework.CleanupTempDir(initialDir, tempDir)

			f := framework.NewDefaultFramework(initialDir + "/bin")
			_ = f.DevPodProviderAdd(ctx, "docker")
			err = f.DevPodProviderUse(context.Background(), "docker")
			framework.ExpectNoError(err)

			ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

			// Start up devpod workspace
			devpodUpDeadline := time.Now().Add(5 * time.Minute)
			devpodUpCtx, cancel := context.WithDeadline(context.Background(), devpodUpDeadline)
			defer cancel()
			err = f.DevPodUp(devpodUpCtx, tempDir)
			framework.ExpectNoError(err)

			devpodSSHDeadline := time.Now().Add(20 * time.Second)
			devpodSSHCtx, cancelSSH := context.WithDeadline(context.Background(), devpodSSHDeadline)
			defer cancelSSH()
			err = f.DevPodSSHEchoTestString(devpodSSHCtx, tempDir)
			framework.ExpectNoError(err)
		})

		// ginkgo.It("should start a new workspace with a docker provider (default) and forward gpg agent into it", func() {
		// 	// skip windows for now
		// 	if runtime.GOOS == "windows" {
		// 		return
		// 	}
		//
		// 	tempDir, err := framework.CopyToTempDir("tests/ssh/testdata/gpg-forwarding")
		// 	framework.ExpectNoError(err)
		// 	defer framework.CleanupTempDir(initialDir, tempDir)
		//
		// 	f := framework.NewDefaultFramework(initialDir + "/bin")
		// 	_ = f.DevPodProviderAdd(ctx, "docker")
		// 	err = f.DevPodProviderUse(context.Background(), "docker")
		// 	framework.ExpectNoError(err)
		//
		// 	ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)
		//
		// 	out, err := exec.Command("gpg", "-k").Output()
		// 	if err != nil || len(out) == 0 {
		// 		err = f.SetupGPG(tempDir)
		// 		framework.ExpectNoError(err)
		// 	}
		//
		// 	// Start up devpod workspace
		// 	devpodUpDeadline := time.Now().Add(5 * time.Minute)
		// 	devpodUpCtx, cancel := context.WithDeadline(context.Background(), devpodUpDeadline)
		// 	defer cancel()
		// 	err = f.DevPodUp(devpodUpCtx, tempDir, "--gpg-agent-forwarding")
		// 	framework.ExpectNoError(err)
		//
		// 	devpodSSHDeadline := time.Now().Add(20 * time.Second)
		// 	devpodSSHCtx, cancelSSH := context.WithDeadline(context.Background(), devpodSSHDeadline)
		// 	defer cancelSSH()
		//
		// 	// GPG agent might be not ready, let's try 10 times, 1 second each
		// 	retries := 10
		// 	for retries > 0 {
		// 		err = f.DevPodSSHGpgTestKey(devpodSSHCtx, tempDir)
		// 		if err != nil {
		// 			retries--
		// 			time.Sleep(time.Second)
		// 		} else {
		// 			break
		// 		}
		// 	}
		// 	framework.ExpectNoError(err)
		// })

		ginkgo.It("should start a new workspace with a docker provider (default) and forward a port into it", func() {
			// skip windows for now
			if runtime.GOOS == "windows" {
				return
			}

			tempDir, err := framework.CopyToTempDir("tests/ssh/testdata/forward-test")
			framework.ExpectNoError(err)
			defer framework.CleanupTempDir(initialDir, tempDir)

			f := framework.NewDefaultFramework(initialDir + "/bin")
			_ = f.DevPodProviderAdd(ctx, "docker")
			err = f.DevPodProviderUse(context.Background(), "docker")
			framework.ExpectNoError(err)

			ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

			// Create a new random number generator with a custom seed (e.g., current time)
			source := rand.NewSource(time.Now().UnixNano())
			rng := rand.New(source)

			// Start up devpod workspace
			devpodUpDeadline := time.Now().Add(5 * time.Minute)
			devpodUpCtx, cancel := context.WithDeadline(context.Background(), devpodUpDeadline)
			defer cancel()
			err = f.DevPodUp(devpodUpCtx, tempDir)
			framework.ExpectNoError(err)

			// Generate a random number for the server port between 50000 and 51000
			port := rng.Intn(1000) + 50000

			fmt.Println("Running netcat server on port", port)

			devpodSSHDeadline := time.Now().Add(20 * time.Second)
			devpodSSHCtx, cancelSSH := context.WithDeadline(context.Background(), devpodSSHDeadline)
			defer cancelSSH()

			fmt.Println("Starting pong service")
			// start a ping/pong service on the port
			cmd := exec.CommandContext(ctx, f.DevpodBinDir+"/"+f.DevpodBinName,
				[]string{
					"ssh", tempDir, "--command",
					"sh -c \"while true; do echo PONG | nc -n -lk -p " + strconv.Itoa(port) + "; done\"",
				}...,
			)
			err = cmd.Start()
			framework.ExpectNoError(err)

			fmt.Println("Forwarding port", port)
			// start ssh with port forwarding in background
			go func() {
				_ = f.DevpodPortTest(devpodSSHCtx, strconv.Itoa(port), tempDir)
			}()

			fmt.Println("Waiting for port", port, "to be open")
			// wait for port to be open
			retries := 5
			out := ""
			address := net.JoinHostPort("localhost", strconv.Itoa(port))
			for retries > 0 {
				fmt.Println("retries left", retries)

				// wait 3s between retries
				time.Sleep(time.Second * 3)

				conn, err := net.Dial("tcp", address)
				if err != nil {
					fmt.Println("port still closed")
					retries = retries - 1
				} else {
					if conn != nil {
						defer conn.Close()
						fmt.Println("connecting to", port)

						fmt.Println("sending PING")
						_, err = conn.Write([]byte("PING"))
						framework.ExpectNoError(err)

						fmt.Println("waiting for response")
						out, err = bufio.NewReader(conn).ReadString('\n')
						if err != nil {
							fmt.Println("invalid response")
							retries = retries - 1
						} else {
							fmt.Println("received", out)

							break
						}
					} else {
						fmt.Println("invalid connection")
						retries = retries - 1
					}
				}
			}
			framework.ExpectNotEqual(retries, 0)

			fmt.Println("Verifying output match")
			framework.ExpectEqual(out, "PONG\n")
			fmt.Println("Success")
		})

		ginkgo.It("should start a new workspace with a docker provider (default) and forward a unix socket into it", func() {
			// unix sockets on windows are not covered by this test
			if runtime.GOOS == "windows" {
				return
			}

			tempDir, err := framework.CopyToTempDir("tests/ssh/testdata/forward-unix-test")
			framework.ExpectNoError(err)
			defer framework.CleanupTempDir(initialDir, tempDir)

			f := framework.NewDefaultFramework(initialDir + "/bin")
			_ = f.DevPodProviderAdd(ctx, "docker")
			err = f.DevPodProviderUse(context.Background(), "docker")
			framework.ExpectNoError(err)

			ginkgo.DeferCleanup(f.DevPodWorkspaceDelete, context.Background(), tempDir)

			// Start up devpod workspace
			devpodUpDeadline := time.Now().Add(5 * time.Minute)
			devpodUpCtx, cancel := context.WithDeadline(context.Background(), devpodUpDeadline)
			defer cancel()
			err = f.DevPodUp(devpodUpCtx, tempDir)
			framework.ExpectNoError(err)

			// Keep the local path short: unix socket paths are limited to ~104 bytes on macOS
			localDir, err := os.MkdirTemp("", "dpsock")
			framework.ExpectNoError(err)
			defer os.RemoveAll(localDir)
			localSocket := filepath.Join(localDir, "pong.sock")
			remoteSocket := "/tmp/devpod-pong.sock"

			devpodSSHDeadline := time.Now().Add(20 * time.Second)
			devpodSSHCtx, cancelSSH := context.WithDeadline(context.Background(), devpodSSHDeadline)
			defer cancelSSH()

			fmt.Println("Starting pong service on", remoteSocket)
			// start a ping/pong service on the remote unix socket
			cmd := exec.CommandContext(ctx, f.DevpodBinDir+"/"+f.DevpodBinName,
				[]string{
					"ssh", tempDir, "--command",
					"socat UNIX-LISTEN:" + remoteSocket + ",fork,unlink-early SYSTEM:'echo PONG'",
				}...,
			)
			err = cmd.Start()
			framework.ExpectNoError(err)

			fmt.Println("Forwarding", localSocket, "to", remoteSocket)
			// start ssh with unix socket forwarding in background
			go func() {
				_ = f.DevpodPortTest(devpodSSHCtx, localSocket+":"+remoteSocket, tempDir)
			}()

			fmt.Println("Waiting for", localSocket, "to answer")
			retries := 5
			out := ""
			for retries > 0 {
				fmt.Println("retries left", retries)

				// wait 3s between retries
				time.Sleep(time.Second * 3)

				conn, err := net.Dial("unix", localSocket)
				if err != nil {
					fmt.Println("socket not ready")
					retries = retries - 1
					continue
				}

				fmt.Println("sending PING")
				_, err = conn.Write([]byte("PING"))
				if err != nil {
					_ = conn.Close()
					fmt.Println("write failed")
					retries = retries - 1
					continue
				}

				fmt.Println("waiting for response")
				out, err = bufio.NewReader(conn).ReadString('\n')
				_ = conn.Close()
				if err != nil {
					// the forward is up but the remote end was refused or not listening yet
					fmt.Println("invalid response")
					retries = retries - 1
					continue
				}

				fmt.Println("received", out)
				break
			}
			framework.ExpectNotEqual(retries, 0)

			fmt.Println("Verifying output match")
			framework.ExpectEqual(out, "PONG\n")
			fmt.Println("Success")
		})
	})
})
