package grpcdemo

import (
	"context"
	"log/slog"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/subrammiles/server_golang/12-rest-api/course"
	coursev1 "github.com/subrammiles/server_golang/19-grpc/gen/course/v1"
)

// newClient runs a real gRPC server over an IN-MEMORY connection (bufconn): no ports, fast, parallel-safe.
func newClient(t *testing.T) coursev1.CourseServiceClient {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	svc := course.NewService(course.NewMemoryRepository(course.Seed()...))
	srv := NewGRPCServer("test-token", slog.New(slog.DiscardHandler))
	coursev1.RegisterCourseServiceServer(srv, NewServer(svc, 10*time.Millisecond))
	go srv.Serve(lis)
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return coursev1.NewCourseServiceClient(conn)
}

func authed() context.Context {
	return metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer test-token")
}

func TestUnaryCalls(t *testing.T) {
	c := newClient(t)
	got, err := c.GetCourse(authed(), &coursev1.GetCourseRequest{Id: 1})
	if err != nil || got.GetCourse().GetTitle() != "Go Fundamentals" || got.GetCourse().GetCreatedAt().AsTime().IsZero() {
		t.Fatalf("got %v %v", got, err)
	}
	list, err := c.ListCourses(authed(), &coursev1.ListCoursesRequest{Category: "backend", PageSize: 1})
	if err != nil || list.GetTotal() != 2 || len(list.GetCourses()) != 1 {
		t.Fatalf("list %v %v", list, err)
	}
	created, err := c.CreateCourse(authed(), &coursev1.CreateCourseRequest{Title: "gRPC", Category: "backend", DurationHours: 5, Price: 499})
	if err != nil || created.GetCourse().GetId() != 4 {
		t.Fatalf("create %v %v", created, err)
	}
}

func TestErrorsBecomeStatusCodes(t *testing.T) {
	c := newClient(t)
	tests := map[string]struct {
		call func() error
		want codes.Code
	}{
		"not found": {func() error { _, err := c.GetCourse(authed(), &coursev1.GetCourseRequest{Id: 99}); return err }, codes.NotFound},
		"bad id":    {func() error { _, err := c.GetCourse(authed(), &coursev1.GetCourseRequest{Id: -1}); return err }, codes.InvalidArgument},
		"invalid":   {func() error { _, err := c.CreateCourse(authed(), &coursev1.CreateCourseRequest{}); return err }, codes.InvalidArgument},
		"duplicate": {func() error {
			_, err := c.CreateCourse(authed(), &coursev1.CreateCourseRequest{Title: "go fundamentals", Category: "x", DurationHours: 1})
			return err
		}, codes.AlreadyExists},
		"no token": {func() error {
			_, err := c.GetCourse(context.Background(), &coursev1.GetCourseRequest{Id: 1})
			return err
		}, codes.Unauthenticated},
		"wrong token": {func() error {
			ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer nope")
			_, err := c.GetCourse(ctx, &coursev1.GetCourseRequest{Id: 1})
			return err
		}, codes.Unauthenticated},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := status.Code(tt.call()); got != tt.want {
				t.Fatalf("code %s, want %s", got, tt.want)
			}
		})
	}
}

func TestServerStreamingUntilDeadline(t *testing.T) {
	c := newClient(t)
	ctx, cancel := context.WithTimeout(authed(), 200*time.Millisecond)
	defer cancel()
	stream, err := c.WatchEnrollments(ctx, &coursev1.WatchEnrollmentsRequest{CourseId: 1})
	if err != nil {
		t.Fatal(err)
	}
	var events []*coursev1.WatchEnrollmentsResponse
	for {
		ev, err := stream.Recv()
		if err != nil {
			if status.Code(err) != codes.DeadlineExceeded {
				t.Fatalf("stream ended with %v", err)
			}
			break
		}
		events = append(events, ev)
	}
	if len(events) < 3 || events[0].GetStudent() != "asha" || events[1].GetSeatsLeft() != 28 {
		t.Fatalf("got %d events: %v", len(events), events)
	}

	// For streaming RPCs, a server error arrives on the first Recv, not on the call itself
	missing, err := c.WatchEnrollments(authed(), &coursev1.WatchEnrollmentsRequest{CourseId: 99})
	if err == nil {
		_, err = missing.Recv()
	}
	if status.Code(err) != codes.NotFound {
		t.Fatalf("unknown course: %v", err)
	}
}
