package data

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"strings"
)

// StoredChannel stores attributes for channels.
type StoredChannel struct {
	NetID string
	Name  string
	JSONStorer
}

// NewStoredChannel creates a new stored channel.
func NewStoredChannel(netID, name string) *StoredChannel {
	return &StoredChannel{netID, name, make(JSONStorer)}
}

// makeID is used to create a key to store this instance by.
func (s *StoredChannel) makeID() string {
	return strings.ToLower(fmt.Sprintf("%s.%s", s.Name, s.NetID))
}

// serialize turns the StoredChannel into bytes for storage.
func (s *StoredChannel) serialize() ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := gob.NewEncoder(buffer)
	err := encoder.Encode(s)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// deserializeChannel reverses the Serialize process.
func deserializeChannel(serialized []byte) (*StoredChannel, error) {
	buffer := &bytes.Buffer{}
	decoder := gob.NewDecoder(buffer)
	if _, err := buffer.Write(serialized); err != nil {
		return nil, err
	}

	dec := &StoredChannel{}
	err := decoder.Decode(dec)
	return dec, err
}
