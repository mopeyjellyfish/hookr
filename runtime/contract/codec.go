package contract

import "context"

// Decoder converts wire bytes into a strongly typed request.
type Decoder[T any] func(payload []byte) (T, error)

// Encoder converts a strongly typed response into wire bytes.
type Encoder[T any] func(value T) ([]byte, error)

// BindHostMethod wraps a typed host callback into a byte-level Handler.
// Generated code should provide the decoder/encoder for its schema/codec.
func BindHostMethod[Req any, Resp any](
	decodeReq Decoder[Req],
	encodeResp Encoder[Resp],
	fn func(ctx context.Context, req Req) (Resp, error),
) Handler {
	return func(ctx context.Context, payload []byte) ([]byte, error) {
		req, err := decodeReq(payload)
		if err != nil {
			return nil, err
		}
		resp, err := fn(ctx, req)
		if err != nil {
			return nil, err
		}
		return encodeResp(resp)
	}
}
