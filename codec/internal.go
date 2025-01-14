package codec

import (
	"encoding"
	"errors"
	"fmt"
	"io"
)

var errNotCustomMarshaler = errors.New("not custom")

// Tries to encode the given data using user-provided marshalers. If the data
// does not implement either encoding.BinaryMarshaler or encoding.TextMarshaler,
// errNotCustomMarshaler is returned.
func encodeCustomMarshaler[T any](w io.Writer, data T) error {
	idata := any(data)

	if m, ok := idata.(encoding.BinaryMarshaler); ok {
		return encodeBinary(w, m)
	}

	if m, ok := idata.(encoding.TextMarshaler); ok {
		return encodeText(w, m)
	}

	return errNotCustomMarshaler
}

func encodeBinary(w io.Writer, m encoding.BinaryMarshaler) error {
	b, err := m.MarshalBinary()
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func encodeText(w io.Writer, m encoding.TextMarshaler) error {
	b, err := m.MarshalText()
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

// Tries to decode the given data using user-provided marshalers. If the data
// does not implement either encoding.BinaryUnmarshaler or encoding.TextUnmarshaler,
// errNotCustomMarshaler is returned.
func decodeCustomMarshaler[T any](r io.Reader, data *T) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reader: %w", err)
	}

	idata := any(data)

	if m, ok := idata.(encoding.BinaryUnmarshaler); ok {
		if err = m.UnmarshalBinary(b); err != nil {
			err = fmt.Errorf("unmarshal binary: %w", err)
		}
		return err
	}

	if m, ok := idata.(encoding.TextUnmarshaler); ok {
		if err = m.UnmarshalText(b); err != nil {
			err = fmt.Errorf("unmarshal text: %w", err)
		}
		return err
	}

	return errNotCustomMarshaler
}
