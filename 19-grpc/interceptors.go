package grpcdemo

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// INTERCEPTORS are gRPC's middleware. Unary ones wrap single calls, stream ones wrap streams.

func LoggingInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		log.Info("rpc", "method", info.FullMethod, "code", status.Code(err).String(), "duration", time.Since(start))
		return resp, err
	}
}

// AuthUnary / AuthStream check a bearer token sent as METADATA (gRPC's headers).
// Real systems verify a JWT here (lesson 16) or use mTLS between services.
func AuthUnary(token string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if err := checkToken(ctx, token); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}

func AuthStream(token string) grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := checkToken(ss.Context(), token); err != nil {
			return err
		}
		return handler(srv, ss)
	}
}

func checkToken(ctx context.Context, want string) error {
	md, _ := metadata.FromIncomingContext(ctx)
	got, _ := strings.CutPrefix(strings.Join(md.Get("authorization"), ""), "Bearer ")
	if subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
		return status.Error(codes.Unauthenticated, "missing or invalid token")
	}
	return nil
}

// NewGRPCServer assembles the server with its interceptor chain.
func NewGRPCServer(token string, log *slog.Logger) *grpc.Server {
	return grpc.NewServer(
		grpc.ChainUnaryInterceptor(LoggingInterceptor(log), AuthUnary(token)),
		grpc.ChainStreamInterceptor(AuthStream(token)),
	)
}
