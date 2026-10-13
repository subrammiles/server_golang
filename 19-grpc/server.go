// Package grpcdemo serves lesson 12's course service over gRPC: binary Protocol Buffers over HTTP/2,
// with a typed contract (proto/course/v1/course.proto) and generated clients for any language.
package grpcdemo

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/subrammiles/server_golang/12-rest-api/course"
	coursev1 "github.com/subrammiles/server_golang/19-grpc/gen/course/v1"
)

// Server implements the generated coursev1.CourseServiceServer interface.
type Server struct {
	coursev1.UnimplementedCourseServiceServer // forward compatibility: new RPCs return Unimplemented
	svc                                       *course.Service
	eventEvery                                time.Duration
}

func NewServer(svc *course.Service, eventEvery time.Duration) *Server {
	return &Server{svc: svc, eventEvery: eventEvery}
}

func (s *Server) GetCourse(ctx context.Context, req *coursev1.GetCourseRequest) (*coursev1.GetCourseResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id must be positive")
	}
	c, err := s.svc.Get(ctx, req.GetId())
	if err != nil {
		return nil, toStatus(err)
	}
	return &coursev1.GetCourseResponse{Course: toProto(c)}, nil
}

func (s *Server) ListCourses(ctx context.Context, req *coursev1.ListCoursesRequest) (*coursev1.ListCoursesResponse, error) {
	size := int(req.GetPageSize())
	if size == 0 {
		size = 10
	}
	page, err := s.svc.List(ctx, course.Query{Category: req.GetCategory(), Page: int(req.GetPage()), Size: size, Sort: "title"})
	if err != nil {
		return nil, toStatus(err)
	}
	out := &coursev1.ListCoursesResponse{Total: int32(page.TotalElements)}
	for _, c := range page.Items {
		out.Courses = append(out.Courses, toProto(c))
	}
	return out, nil
}

func (s *Server) CreateCourse(ctx context.Context, req *coursev1.CreateCourseRequest) (*coursev1.CreateCourseResponse, error) {
	c, err := s.svc.Create(ctx, course.Input{Title: req.GetTitle(), Category: req.GetCategory(),
		DurationHours: int(req.GetDurationHours()), Price: req.GetPrice()})
	if err != nil {
		return nil, toStatus(err)
	}
	return &coursev1.CreateCourseResponse{Course: toProto(c)}, nil
}

// WatchEnrollments is SERVER STREAMING: it keeps sending events until the client cancels (or the deadline passes).
func (s *Server) WatchEnrollments(req *coursev1.WatchEnrollmentsRequest, stream coursev1.CourseService_WatchEnrollmentsServer) error {
	if _, err := s.svc.Get(stream.Context(), req.GetCourseId()); err != nil {
		return toStatus(err)
	}
	students := []string{"asha", "ravi", "meera", "kiran"}
	ticker := time.NewTicker(s.eventEvery)
	defer ticker.Stop()
	for i := 0; ; i++ {
		select {
		case <-stream.Context().Done():
			return nil // client went away: stop working
		case <-ticker.C:
			ev := &coursev1.WatchEnrollmentsResponse{CourseId: req.GetCourseId(), Student: students[i%len(students)], SeatsLeft: int32(30 - i - 1)}
			if err := stream.Send(ev); err != nil {
				return err
			}
		}
	}
}

// toStatus maps domain errors to gRPC STATUS CODES (gRPC's equivalent of HTTP status codes).
func toStatus(err error) error {
	var ve *course.ValidationError
	switch {
	case errors.As(err, &ve):
		return status.Errorf(codes.InvalidArgument, "invalid fields: %v", ve.Fields)
	case errors.Is(err, course.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, course.ErrDuplicate):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, err.Error())
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}

func toProto(c course.Course) *coursev1.Course {
	return &coursev1.Course{Id: c.ID, Title: c.Title, Category: c.Category,
		DurationHours: int32(c.DurationHours), Price: c.Price, CreatedAt: timestamppb.New(c.CreatedAt)}
}
