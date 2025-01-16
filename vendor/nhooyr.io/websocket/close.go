//go:build !js
// +build !js

package websocket

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"

	"nhooyr.io/websocket/internal/errd"
)

// StatusCode represents a WebSocket status code.
// https://tools.ietf.org/html/rfc6455#section-7.4
type StatusCode int

// https://www.iana.org/assignments/websocket/websocket.xhtml#close-code-number
//
// These are only the status codes defined by the protocol.
//
// You can define custom codes in the 3000-4999 range.
// The 3000-3999 range is reserved for use by libraries, frameworks and applications.
// The 4000-4999 range is reserved for private use.
const (
	StatusNormalClosure   StatusCode = 1000
	StatusGoingAway       StatusCode = 1001
	StatusProtocolError   StatusCode = 1002
	StatusUnsupportedData StatusCode = 1003

	// 1004 is reserved and so unexported.
	statusReserved StatusCode = 1004

	// StatusNoStatusRcvd cannot be sent in a close message.
	// It is reserved for when a close message is received without
	// a status code.
	StatusNoStatusRcvd StatusCode = 1005

	// StatusAbnormalClosure is exported for use only with Wasm.
	// In non Wasm Go, the returned error will indicate whether the
	// connection was closed abnormally.
	StatusAbnormalClosure StatusCode = 1006

	StatusInvalidFramePayloadData StatusCode = 1007
	StatusPolicyViolation         StatusCode = 1008
	StatusMessageTooBig           StatusCode = 1009
	StatusMandatoryExtension      StatusCode = 1010
	StatusInternalError           StatusCode = 1011
	StatusServiceRestart          StatusCode = 1012
	StatusTryAgainLater           StatusCode = 1013
	StatusBadGateway              StatusCode = 1014

	// StatusTLSHandshake is only exported for use with Wasm.
	// In non Wasm Go, the returned error will indicate whether there was
	// a TLS handshake failure.
	StatusTLSHandshake StatusCode = 1015
)

// CloseError is returned when the connection is closed with a status and reason.
//
// Use Go 1.13's errors.As to check for this error.
// Also see the CloseStatus helper.
type CloseError struct {
	Code   StatusCode
	Reason string
}

func (ce CloseError) Error() string {
	return fmt.Sprintf("status = %v and reason = %q", ce.Code, ce.Reason)
}

// CloseStatus is a convenience wrapper around Go 1.13's errors.As to grab
// the status code from a CloseError.
//
// -1 will be returned if the passed error is nil or not a CloseError.
func CloseStatus(err error) StatusCode {
	var ce CloseError
	if errors.As(err, &ce) {
		return ce.Code
	}
	return -1
}

// Close performs the WebSocket close handshake with the given status code and reason.
//
// It will write a WebSocket close frame with a timeout of 5s and then wait 5s for
// the peer to send a close frame.
// All data messages received from the peer during the close handshake will be discarded.
//
// The connection can only be closed once. Additional calls to Close
// are no-ops.
//
// The maximum length of reason must be 125 bytes. Avoid
// sending a dynamic reason.
//
// Close will unblock all goroutines interacting with the connection once
// complete.
func (c *Conn) Close(code StatusCode, reason string) error {
	defer c.wg.Wait()
	return c.closeHandshake(code, reason)
}

// CloseNow closes the WebSocket connection without attempting a close handshake.
// Use when you do not want the overhead of the close handshake.
func (c *Conn) CloseNow() (err error) {
	defer c.wg.Wait()
	defer errd.Wrap(&err, "failed to close WebSocket")

	if c.isClosed() {
		return net.ErrClosed
	}

	c.close(nil)
	return c.closeErr
}

func (c *Conn) closeHandshake(code StatusCode, reason string) (err error) {
	defer errd.Wrap(&err, "failed to close WebSocket")

	writeErr := c.writeClose(code, reason)
	closeHandshakeErr := c.waitCloseHandshake()

	if writeErr != nil {
		return writeErr
	}

	if CloseStatus(closeHandshakeErr) == -1 && !errors.Is(net.ErrClosed, closeHandshakeErr) {
		return closeHandshakeErr
	}

	return nil
}

func (c *Conn) writeClose(code StatusCode, reason string) error {
	c.closeMu.Lock()
	wroteClose := c.wroteClose
	c.wroteClose = true
	c.closeMu.Unlock()
	if wroteClose {
		return net.ErrClosed
	}

	ce := CloseError{
		Code:   code,
		Reason: reason,
	}

	var p []byte
	var marshalErr error
	if ce.Code != StatusNoStatusRcvd {
		p, marshalErr = ce.bytes()
	}

	writeErr := c.writeControl(context.Background(), opClose, p)
	if CloseStatus(writeErr) != -1 {
		// Not a real error if it's due to a close frame being received.
		writeErr = nil
	}

	// We do this after in case there was an error writing the close frame.
	c.setCloseErr(fmt.Errorf("sent close frame: %w", ce))

	if marshalErr != nil {
		return marshalErr
	}
	return writeErr
}

func (c *Conn) waitCloseHandshake() error {
	defer c.close(nil)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err := c.readMu.lock(ctx)
	if err != nil {
		return err
	}
	defer c.readMu.unlock()

	if c.readCloseFrameErr != nil {
		return c.readCloseFrameErr
	}

	for i := int64(0); i < c.msgReader.payloadLength; i++ {
		_, err := c.br.ReadByte()
		if err != nil {
			return err
		}
	}

	for {
		h, err := c.readLoop(ctx)
		if err != nil {
			return err
		}

		for i := int64(0); i < h.payloadLength; i++ {
			_, err := c.br.ReadByte()
			if err != nil {
				return err
			}
		}
	}
}

func parseClosePayload(p []byte) (CloseError, error) {
	if len(p) == 0 {
		return CloseError{
			Code: StatusNoStatusRcvd,
		}, nil
	}

	if len(p) < 2 {
		return CloseError{}, fmt.Errorf("close payload %q too small, cannot even contain the 2 byte status code", p)
	}

	ce := CloseError{
		Code:   StatusCode(binary.BigEndian.Uint16(p)),
		Reason: string(p[2:]),
	}

	if !validWireCloseCode(ce.Code) {
		return CloseError{}, fmt.Errorf("invalid status code %v", ce.Code)
	}

	return ce, nil
}

// See http://www.iana.org/assignments/websocket/websocket.xhtml#close-code-number
// and https://tools.ietf.org/html/rfc6455#section-7.4.1
func validWireCloseCode(code StatusCode) bool {
	switch code {
	case statusReserved, StatusNoStatusRcvd, StatusAbnormalClosure, StatusTLSHandshake:
		return false
	}

	if code >= StatusNormalClosure && code <= StatusBadGateway {
		return true
	}
	if code >= 3000 && code <= 4999 {
		return true
	}

	return false
}

func (ce CloseError) bytes() ([]byte, error) {
	p, err := ce.bytesErr()
	if err != nil {
		err = fmt.Errorf("failed to marshal close frame: %w", err)
		ce = CloseError{
			Code: StatusInternalError,
		}
		p, _ = ce.bytesErr()
	}
	return p, err
}

const maxCloseReason = maxControlPayload - 2

func (ce CloseError) bytesErr() ([]byte, error) {
	if len(ce.Reason) > maxCloseReason {
		return nil, fmt.Errorf("reason string max is %v but got %q with length %v", maxCloseReason, ce.Reason, len(ce.Reason))
	}

	if !validWireCloseCode(ce.Code) {
		return nil, fmt.Errorf("status code %v cannot be set", ce.Code)
	}

	buf := make([]byte, 2+len(ce.Reason))
	binary.BigEndian.PutUint16(buf, uint16(ce.Code))
	copy(buf[2:], ce.Reason)
	return buf, nil
}

func (c *Conn) setCloseErr(err error) {
	c.closeMu.Lock()
	c.setCloseErrLocked(err)
	c.closeMu.Unlock()
}

func (c *Conn) setCloseErrLocked(err error) {
	if c.closeErr == nil && err != nil {
		c.closeErr = fmt.Errorf("WebSocket closed: %w", err)
	}
}

func (c *Conn) isClosed() bool {
	select {
	case <-c.closed:
		return true
	default:
		return false
	}
}
