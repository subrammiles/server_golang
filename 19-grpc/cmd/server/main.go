// go run ./19-grpc/cmd/server     then in another terminal: go run ./19-grpc/cmd/client
package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/subrammiles/server_golang/12-rest-api/course"
	grpcdemo "github.com/subrammiles/server_golang/19-grpc"
	coursev1 "github.com/subrammiles/server_golang/19-grpc/gen/course/v1"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	svc := course.NewService(course.NewMemoryRepository(course.Seed()...))
	srv := grpcdemo.NewGRPCServer("demo-token", log)

	coursev1.RegisterCourseServiceServer(srv, grpcdemo.NewServer(svc, time.Second))
	healthpb.RegisterHealthServer(srv, health.NewServer()) // standard health checks (Kubernetes probes)
	reflection.Register(srv)                               // lets tools like grpcurl discover the API

	ln, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Error("listen", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		srv.GracefulStop() // finish in-flight RPCs
	}()
	log.Info("gRPC server on :9090")
	if err := srv.Serve(ln); err != nil {
		log.Error("serve", "error", err)
	}
}
