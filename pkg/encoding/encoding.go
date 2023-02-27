package encoding

//go:generate protoc -I ../../protobuf --go_opt=paths=source_relative --go_out=. abyssfile.proto

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"

	"google.golang.org/protobuf/proto"
)

// Encode AbyssRecording into io.WriteCloser
func (rf *AbyssRecording) Encode(w io.Writer) error {
	data, err := proto.Marshal(rf)
	if err != nil {
		return err
	}

	gw := gzip.NewWriter(w)
	defer gw.Close()

	_, err = io.Copy(gw, bytes.NewReader(data))

	return err
}

// Decode abyss file back into AbyssRecording struct
func Decode(r io.Reader) (*AbyssRecording, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	result := &AbyssRecording{}

	data, err := ioutil.ReadAll(gr)
	if err != nil {
		return result, err
	}

	err = proto.Unmarshal(data, result)

	return result, err
}
