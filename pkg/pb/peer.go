package pb

import "io"

type Peer interface {
	WritePeerConfig(io.Writer) error
}
