package signal

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
)

var (
	onlyOneSignalHandler = make(chan struct{})
	shutdownHandler      chan os.Signal
	shutdownSignals      = []os.Signal{os.Interrupt, syscall.SIGTERM}
)

// SetupSignalContext is same as SetupSignalHandler, but a context.Context is returned.
// Only one of SetupSignalContext and SetupSignalHandler should be called, and only can
// be called once.
func SetupSignalContext() context.Context {
	close(onlyOneSignalHandler) // panics when called twice

	shutdownHandler = make(chan os.Signal, 2)

	ctx, cancel := context.WithCancel(context.Background())
	signal.Notify(shutdownHandler, shutdownSignals...)
	go func() {
		s := <-shutdownHandler
		cancel()
		fmt.Println()
		logrus.Warnf("Abort: [%s] received, cleaning up resources", s.String())
		logrus.Warnf("Use 'Ctrl-C' again to force exit (not recommended)")
		<-shutdownHandler

		// second signal. Exit directly.
		logrus.Warnf("Hangar was forced to stop.")
		if err := os.RemoveAll(utils.HangarCacheDir()); err != nil {
			logrus.Warnf("failed to delete %q: %v", utils.HangarCacheDir(), err)
		}
		os.Exit(130)
	}()

	return ctx
}
