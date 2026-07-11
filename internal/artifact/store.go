package artifact

import "context"

type PutResult struct {
	URI      string
	Path     string
	SHA256   string
	ByteSize int64
}

type Store interface {
	Put(ctx context.Context, key string, content []byte) (PutResult, error)
}
