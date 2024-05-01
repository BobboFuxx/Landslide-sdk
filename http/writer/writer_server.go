package writer

import (
	"context"
	"io"

	writerpb "github.com/consideritdone/landslidevm/proto/io/writer"
)

var _ writerpb.WriterServer = (*Server)(nil)

// Server is an http.Handler that is managed over RPC.
type Server struct {
	writerpb.UnsafeWriterServer
	writer io.Writer
}

// NewServer returns an http.Handler instance managed remotely
func NewServer(writer io.Writer) *Server {
	return &Server{writer: writer}
}

func (s *Server) Write(_ context.Context, req *writerpb.WriteRequest) (*writerpb.WriteResponse, error) {
	n, err := s.writer.Write(req.Payload)
	resp := &writerpb.WriteResponse{
		Written: int32(n),
	}
	if err != nil {
		errStr := err.Error()
		resp.Error = &errStr
	}
	return resp, nil
}
