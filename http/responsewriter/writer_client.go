package responsewriter

import (
	"bufio"
	"context"
	"net"
	"net/http"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/consideritdone/landslidevm/grpcutils"
	"github.com/consideritdone/landslidevm/http/conn"
	"github.com/consideritdone/landslidevm/http/reader"
	"github.com/consideritdone/landslidevm/http/writer"

	responsewriterpb "github.com/consideritdone/landslidevm/proto/http/responsewriter"
	readerpb "github.com/consideritdone/landslidevm/proto/io/reader"
	writerpb "github.com/consideritdone/landslidevm/proto/io/writer"
	connpb "github.com/consideritdone/landslidevm/proto/net/conn"
)

var (
	_ http.ResponseWriter = (*Client)(nil)
	_ http.Flusher        = (*Client)(nil)
	_ http.Hijacker       = (*Client)(nil)
)

// Client is an http.ResponseWriter that talks over RPC.
type Client struct {
	client responsewriterpb.WriterClient
	header http.Header
}

// NewClient returns a response writer connected to a remote response writer
func NewClient(header http.Header, client responsewriterpb.WriterClient) *Client {
	return &Client{
		client: client,
		header: header,
	}
}

func (c *Client) Header() http.Header {
	return c.header
}

func (c *Client) Write(payload []byte) (int, error) {
	req := &responsewriterpb.WriteRequest{
		Headers: make([]*responsewriterpb.Header, 0, len(c.header)),
		Payload: payload,
	}
	for key, values := range c.header {
		req.Headers = append(req.Headers, &responsewriterpb.Header{
			Key:    key,
			Values: values,
		})
	}
	resp, err := c.client.Write(context.Background(), req)
	if err != nil {
		return 0, err
	}
	return int(resp.Written), nil
}

func (c *Client) WriteHeader(statusCode int) {
	req := &responsewriterpb.WriteHeaderRequest{
		Headers:    make([]*responsewriterpb.Header, 0, len(c.header)),
		StatusCode: int32(statusCode),
	}
	for key, values := range c.header {
		req.Headers = append(req.Headers, &responsewriterpb.Header{
			Key:    key,
			Values: values,
		})
	}
	// TODO: Is there a way to handle the error here?
	_, _ = c.client.WriteHeader(context.Background(), req)
}

func (c *Client) Flush() {
	// TODO: is there a way to handle the error here?
	_, _ = c.client.Flush(context.Background(), &emptypb.Empty{})
}

type addr struct {
	network string
	str     string
}

func (a *addr) Network() string {
	return a.network
}

func (a *addr) String() string {
	return a.str
}

func (c *Client) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	resp, err := c.client.Hijack(context.Background(), &emptypb.Empty{})
	if err != nil {
		return nil, nil, err
	}

	clientConn, err := grpcutils.Dial(resp.ServerAddr)
	if err != nil {
		return nil, nil, err
	}

	conn := conn.NewClient(
		connpb.NewConnClient(clientConn),
		&addr{
			network: resp.LocalNetwork,
			str:     resp.LocalString,
		},
		&addr{
			network: resp.RemoteNetwork,
			str:     resp.RemoteString,
		},
		clientConn,
	)

	reader := reader.NewClient(readerpb.NewReaderClient(clientConn))
	writer := writer.NewClient(writerpb.NewWriterClient(clientConn))

	readWriter := bufio.NewReadWriter(
		bufio.NewReader(reader),
		bufio.NewWriter(writer),
	)

	return conn, readWriter, nil
}