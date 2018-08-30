package wuploader

import (
	"context"
	"errors"
	"sync"

	"github.com/gammazero/nexus/client"
	"github.com/gammazero/nexus/transport/serialize"
	"github.com/gammazero/nexus/wamp"
)

func returnError(err wamp.URI, args ...interface{}) *client.InvokeResult {
	return &client.InvokeResult{
		Err:  err,
		Args: args,
	}
}
func returnResult(args ...interface{}) *client.InvokeResult {
	return &client.InvokeResult{
		Args: args,
	}
}

// UploadChecker is a callback type which is executed to verify the beginning of a transaction
type UploadChecker func(uploadSize int64, args wamp.List, kwargs, details wamp.Dict) error

// UploadHandler is a callback type which is executed when the file upload has finished and the transaction should be executed.
type UploadHandler func(ctx context.Context, uploaded serialize.BinaryData, args wamp.List, kwargs, details wamp.Dict) *client.InvokeResult

// Uploader is the main interface, allowing the user to create and delete data upload endpoints
type Uploader interface {
	Add(endpoint string, begin UploadChecker, handler UploadHandler) error
	Destroy(endpoint string) error
	Stop()
}

type uploadTxn struct {
	buf  serialize.BinaryData
	pos  uint64
	size uint64
}
type uploaderImpl struct {
	client *client.Client
	txns   map[wamp.ID]*uploadTxn
	lock   sync.RWMutex
}

func (u *uploaderImpl) Add(endpoint string, begin UploadChecker, handler UploadHandler) error {
	return u.client.Register(endpoint, func(ctx context.Context, args wamp.List, kwargs, details wamp.Dict) *client.InvokeResult {
		if len(args) == 0 {
			return returnError(wamp.ErrInvalidArgument)
		}
		act, ok := wamp.AsString(args[0])
		if !ok {
			return returnError(wamp.ErrInvalidArgument)
		}
		switch act {
		case "start":
			if len(args) < 2 {
				return returnError(wamp.ErrInvalidArgument)
			}
			uploadSize, ok := wamp.AsInt64(args[1])
			// Files of length 0 will be considered empty files, so allow them being uploaded.
			// For files of zero length, we don't need to initialize any storage
			// Since in go 'nil' is considered equal a zero-length array
			if !ok || uploadSize < 0 {
				return returnError(wamp.ErrInvalidArgument)
			}
			if begin != nil {
				if err := begin(uploadSize, args[2:], kwargs, details); err != nil {
					return returnError("com.robulab.internal-error", err.Error())
				}
			}
			id := wamp.GlobalID()
			u.lock.Lock()
			defer u.lock.Unlock()
			u.txns[id] = &uploadTxn{
				buf:  make([]byte, 0, uploadSize),
				pos:  0,
				size: uint64(uploadSize),
			}
			return returnResult(id)
		case "data":
			if len(args) < 3 {
				return returnError(wamp.ErrInvalidArgument)
			}
			txn, tok := wamp.AsID(args[1])
			data, bok := args[2].(serialize.BinaryData)
			if !tok || !bok {
				return returnError(wamp.ErrInvalidArgument)
			}
			u.lock.RLock()
			txnObj, ok := u.txns[txn]
			u.lock.RUnlock()
			if !ok {
				return returnError("com.robulab.no-such-upload", txn)
			}
			if txnObj.pos+uint64(len(data)) > txnObj.size {
				return returnError("com.robulab.invalid-upload-size")
			}
			txnObj.buf = append(txnObj.buf, data...)
			txnObj.pos += uint64(len(data))
			return returnResult(txnObj.pos)
		case "finish":
			if len(args) < 2 {
				return returnError(wamp.ErrInvalidArgument)
			}
			txn, tok := wamp.AsID(args[1])
			if !tok {
				return returnError(wamp.ErrInvalidArgument)
			}
			u.lock.Lock()
			txnObj, ok := u.txns[txn]
			delete(u.txns, txn)
			u.lock.Unlock()
			if !ok {
				return returnError("com.robulab.no-such-upload", txn)
			}
			if txnObj.pos != txnObj.size || txnObj.pos != uint64(len(txnObj.buf)) {
				return returnError("com.robulab.invalid-upload-size")
			}
			return handler(ctx, txnObj.buf, args[2:], kwargs, details)
		}
		return &client.InvokeResult{
			Err: wamp.ErrNoSuchProcedure,
		}
	}, wamp.Dict{
		wamp.OptDiscloseCaller: true,
		wamp.OptInvoke:         wamp.InvokeSingle, // Single invocation to ensure it will fail!
	})
}
func (u *uploaderImpl) Destroy(endpoint string) error {
	return u.client.Unregister(string(endpoint))
}

func (u *uploaderImpl) Stop() {}

// NewUploader creates a new uploader and starts monitoring tasks.
// Free the returned uploader after use using Uploader.Stop()
func NewUploader(c *client.Client) (Uploader, error) {
	if c == nil {
		return nil, errors.New("client may not be nil")
	}
	return &uploaderImpl{
		client: c,
		txns:   map[wamp.ID]*uploadTxn{},
		lock:   sync.RWMutex{},
	}, nil
}
