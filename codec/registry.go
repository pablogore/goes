package codec

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
)

var (
	// ErrNotFound is returned when trying to encode/decode data which hasn't
	// been registered into a registry.
	ErrNotFound = errors.New("encoding not found. forgot to register?")

	// ErrMissingFactory is returned when trying to instantiate data for which
	// no factory function was provided.
	ErrMissingFactory = errors.New("missing factory for data. forgot to register?")
)

// A Registry provides the Encoders and Decoders for event data or command
// payloads. Use the Register method to register the Encoder and Decoder for a
// specific type.
//
// You likely don't want to use this registry directly, as it requires you to
// define an Encoder and Decoder for every registered type/name. You can for
// example wrap this *Registry in a *GobRegistry to use encoding/gob for
// encoding and decoding data:
//
// Register
//
//	type fooData struct { ... }
//	reg := Gob(New())
//	reg.GobRegister("foo", func() any { return fooData{}})
//
// Encode
//
//	var w io.Writer
//	err := reg.Encode(w, "foo", someData{...})
//
// Decode
//
//	var r io.Reader
//	err := reg.Decode(r, "foo")
type Registry struct {
	sync.RWMutex

	encoders  map[string]Encoder[any]
	decoders  map[string]Decoder[any]
	factories map[string]func() any
}

// Make creates and returns a new instance of the data that is registered under
// the given name. If no factory function was provided for this data,
// ErrMissingFactory is returned. If the data cannot be casted to D, an error
// is returned.
func Make[D any](r *Registry, name string) (D, error) {
	var zero D

	r.RLock()
	defer r.RUnlock()

	if makeFunc, ok := r.factories[name]; ok && makeFunc != nil {
		d := makeFunc()
		if v, ok := d.(D); ok {
			return v, nil
		} else {
			return zero, fmt.Errorf("cannot cast %T to %T", d, v)
		}
	}

	return zero, ErrMissingFactory
}

// Register registers the encoding for events with the given name.
func Register[D any, Enc Encoder[D], Dec Decoder[D]](r *Registry, name string, enc Enc, dec Dec) {
	registerWithFactoryFunc[D](r, name, enc, dec, func() any {
		var v D
		return v
	})
}

func registerWithFactoryFunc[D any, Enc Encoder[D], Dec Decoder[D]](r *Registry, name string, enc Enc, dec Dec, fn func() any) {
	r.Lock()
	defer r.Unlock()

	r.encoders[name] = EncoderFunc[any](func(w io.Writer, data any) error {
		return enc.Encode(w, data.(D))
	})

	r.decoders[name] = DecoderFunc[any](func(r io.Reader) (any, error) {
		return dec.Decode(r)
	})

	r.factories[name] = fn
}

// Encode encodes the data that is registered under the given name using the
// registered Encoder. If no Encoder is registered for the given name, an error
// that unwraps to ErrNotFound is returned.
func Encode[D any](r *Registry, w io.Writer, name string, data D) error {
	r.RLock()
	defer r.RUnlock()

	if err := encodeCustomMarshaler(w, data); !errors.Is(err, errNotCustomMarshaler) {
		return err
	}

	if enc, ok := r.encoders[name]; ok {
		return enc.Encode(w, data)
	}

	return fmt.Errorf("get encoder: %w [name=%v]", ErrNotFound, name)
}

// Decode decodes the data that is registered under the given name using the
// registered Decoder. If no Decoder is registered for the give name, an error
// that unwraps to ErrNotFound is returned.
func Decode[D any](r *Registry, in io.Reader, name string) (D, error) {
	var zero D

	r.RLock()
	defer r.RUnlock()

	if _, ok := r.factories[name]; ok {
		data, err := Make[D](r, name)
		if err != nil {
			return zero, err
		}

		var buf bytes.Buffer
		in = io.TeeReader(in, &buf)

		if err := decodeCustomMarshaler(in, &data); err != errNotCustomMarshaler {
			if err != nil {
				err = fmt.Errorf("custom unmarshaler: %w", err)
			}
			return data, err
		}

		in = &buf
	}

	if dec, ok := r.decoders[name]; ok {
		decoded, err := dec.Decode(in)
		if err != nil {
			return zero, err
		}
		return decoded.(D), nil
	}

	return zero, fmt.Errorf("get decoder: %w [name=%v]", ErrNotFound, name)
}

// New returns a new Registry.
func New() *Registry {
	return &Registry{
		encoders:  make(map[string]Encoder[any]),
		decoders:  make(map[string]Decoder[any]),
		factories: make(map[string]func() any),
	}
}

// // Register registers the given Encoder and Decoder under the given name.
// // When reg.Encode is called, the provided Encoder is be used to encode the
// // given data. When reg.Decode is called, the provided Decoder is used. The
// // makeFunc is required for custom data unmarshalers to work.
// func (reg *Registry) Register(name string, enc Encoder[any], dec Decoder[any], val any) {
// 	Register(reg, name, val, enc, dec)
// }

// Encode encodes the data that is registered under the given name using the
// registered Encoder. If no Encoder is registered for the given name, an error
// that unwraps to ErrNotFound is returned.
func (reg *Registry) Encode(w io.Writer, name string, data any) error {
	return Encode(reg, w, name, data)
}

// Decode decodes the data that is registered under the given name using the
// registered Decoder. If no Decoder is registered for the give name, an error
// that unwraps to ErrNotFound is returned.
func (reg *Registry) Decode(r io.Reader, name string) (any, error) {
	return Decode[any](reg, r, name)
}

// New creates and returns a new instance of the data that is registered under
// the given name. If no factory function was provided for this data,
// ErrMissingFactory is returned.
func (reg *Registry) New(name string) (any, error) {
	return Make[any](reg, name)
}
