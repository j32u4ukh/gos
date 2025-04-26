package tcp_server

import (
	"github.com/j32u4ukh/gos/cntr"
	"github.com/pkg/errors"
)

func FormData(bd *cntr.BinaryData) error {
	err := bd.InsertUInt32(bd.GetLength())
	if err != nil {
		return errors.Wrap(err, "Failed to form data")
	}
	return nil
}
