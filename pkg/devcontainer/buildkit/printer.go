package buildkit

import (
	"context"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/util/progress/progressui"
	"github.com/moby/buildkit/util/progress/progresswriter"
	"io"
)

type printer struct {
	status chan *client.SolveStatus
	done   <-chan struct{}
	err    error
}

func (p *printer) Done() <-chan struct{} {
	return p.done
}

func (p *printer) Err() error {
	return p.err
}

func (p *printer) Status() chan *client.SolveStatus {
	if p == nil {
		return nil
	}
	return p.status
}

func NewPrinter(ctx context.Context, out io.Writer) (progresswriter.Writer, error) {
	statusCh := make(chan *client.SolveStatus)
	doneCh := make(chan struct{})

	pw := &printer{
		status: statusCh,
		done:   doneCh,
	}
	go func() {
		// not using shared context to not disrupt display but let is finish reporting errors
		_, pw.err = progressui.DisplaySolveStatus(ctx, "", nil, out, statusCh)
		close(doneCh)
	}()
	return pw, nil
}
