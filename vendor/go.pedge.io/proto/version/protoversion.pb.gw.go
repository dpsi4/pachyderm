// Code generated by protoc-gen-grpc-gateway
// source: version/protoversion.proto
// DO NOT EDIT!

package protoversion

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gengo/grpc-gateway/runtime"
	"github.com/gengo/grpc-gateway/utilities"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"go.pedge.io/pb/go/google/protobuf"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var _ codes.Code
var _ io.Reader
var _ = runtime.String
var _ = json.Marshal
var _ = utilities.PascalFromSnake

func request_API_GetVersion_0(ctx context.Context, client APIClient, req *http.Request, pathParams map[string]string) (proto.Message, error) {
	var protoReq google_protobuf.Empty

	return client.GetVersion(ctx, &protoReq)
}

// RegisterAPIHandlerFromEndpoint is same as RegisterAPIHandler but
// automatically dials to "endpoint" and closes the connection when "ctx" gets done.
func RegisterAPIHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string) (err error) {
	conn, err := grpc.Dial(endpoint, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if cerr := conn.Close(); cerr != nil {
				glog.Errorf("Failed to close conn to %s: %v", endpoint, cerr)
			}
			return
		}
		go func() {
			<-ctx.Done()
			if cerr := conn.Close(); cerr != nil {
				glog.Errorf("Failed to close conn to %s: %v", endpoint, cerr)
			}
		}()
	}()

	return RegisterAPIHandler(ctx, mux, conn)
}

// RegisterAPIHandler registers the http handlers for service API to "mux".
// The handlers forward requests to the grpc endpoint over "conn".
func RegisterAPIHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	client := NewAPIClient(conn)

	mux.Handle("GET", pattern_API_GetVersion_0, func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
		resp, err := request_API_GetVersion_0(runtime.AnnotateContext(ctx, req), client, req, pathParams)
		if err != nil {
			runtime.HTTPError(ctx, w, req, err)
			return
		}

		forward_API_GetVersion_0(ctx, w, req, resp, mux.GetForwardResponseOptions()...)

	})

	return nil
}

var (
	pattern_API_GetVersion_0 = runtime.MustPattern(runtime.NewPattern(1, []int{2, 0}, []string{"version"}, ""))
)

var (
	forward_API_GetVersion_0 = runtime.ForwardResponseMessage
)
