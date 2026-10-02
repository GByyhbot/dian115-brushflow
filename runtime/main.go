//go:build wasip1 && wasm

// Reactor ABI follows the pinned DIAN115 public complete-plugin example.
package main

import (
	"dian115-brushflow/app"
	"dian115-brushflow/host"
	"encoding/json"
	"errors"
	"runtime"
	"time"
	"unsafe"
)

const frameLimit = 16 << 20

//go:wasmimport dian115 host_call
func hostCall(ptr, length uint32) uint32

//go:wasmimport dian115 host_read
func hostRead(ptr, length uint32) uint32

type broker struct{}

func (broker) Call(method string, params any, out any) error {
	b, err := json.Marshal(map[string]any{"method": method, "params": params})
	if err != nil {
		return err
	}
	size := hostCall(uint32(uintptr(unsafe.Pointer(&b[0]))), uint32(len(b)))
	runtime.KeepAlive(b)
	if size == 0 || size > 2<<20 {
		return errors.New("host response exceeds plugin budget")
	}
	response := make([]byte, size)
	if hostRead(uint32(uintptr(unsafe.Pointer(&response[0]))), size) != size {
		return errors.New("host read failed")
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if err = json.Unmarshal(response, &envelope); err != nil {
		return err
	}
	if len(envelope.Error) > 0 && string(envelope.Error) != "null" {
		return errors.New("host RPC failed")
	}
	return json.Unmarshal(envelope.Result, out)
}

var inputBuffer, outputBuffer []byte
var hostClient = host.Client{Caller: broker{}}
var service = app.Service{Store: hostClient, Broker: hostClient}

//go:wasmexport dian115_alloc
func allocate(size uint32) uint32 {
	if size == 0 || size > frameLimit {
		panic("invalid allocation")
	}
	inputBuffer = make([]byte, size)
	return uint32(uintptr(unsafe.Pointer(&inputBuffer[0])))
}

//go:wasmexport dian115_handle
func handle(ptr, length uint32) uint64 {
	if len(inputBuffer) == 0 || ptr != uint32(uintptr(unsafe.Pointer(&inputBuffer[0]))) || length == 0 || int(length) > len(inputBuffer) {
		panic("invalid invocation buffer")
	}
	result, err := dispatch(inputBuffer[:length])
	envelope := map[string]any{"result": result}
	if err != nil {
		envelope = map[string]any{"error": map[string]any{"code": -32602, "message": err.Error()}}
	}
	outputBuffer, _ = json.Marshal(envelope)
	if len(outputBuffer) > 256<<10 {
		outputBuffer = []byte(`{"error":{"code":-32603,"message":"response exceeds 256 KiB"}}`)
	}
	return uint64(uintptr(unsafe.Pointer(&outputBuffer[0])))<<32 | uint64(len(outputBuffer))
}
func dispatch(raw []byte) (any, error) {
	var m struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	switch m.Method {
	case "runtime.initialize":
		var p struct {
			Protocol string `json:"protocol"`
		}
		if json.Unmarshal(m.Params, &p) != nil || p.Protocol != "dian115:wasm@1" {
			return nil, errors.New("unsupported protocol")
		}
		return map[string]any{"ready": true, "protocol": p.Protocol}, nil
	case "runtime.shutdown":
		return map[string]any{"stopping": true}, nil
	case "runtime.invoke":
		var in app.Input
		if err := json.Unmarshal(m.Params, &in); err != nil {
			return nil, err
		}
		if in.Envelope.Op == "resident" {
			for {
				_ = service.Tick(time.Now())
				time.Sleep(time.Minute)
			}
		}
		return service.Invoke(in)
	default:
		return nil, errors.New("unsupported method")
	}
}
func main() {}
