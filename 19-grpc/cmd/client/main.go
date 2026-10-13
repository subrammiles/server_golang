// A gRPC client using the GENERATED stub. Run the server first: go run ./19-grpc/cmd/server
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	coursev1 "github.com/subrammiles/server_golang/19-grpc/gen/course/v1"
)

func main() {
	// insecure = no TLS, for local demos only. Production uses TLS (or mTLS between services).
	conn, err := grpc.NewClient("localhost:9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	client := coursev1.NewCourseServiceClient(conn)

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer demo-token")
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second) // ALWAYS set a deadline on RPCs
	defer cancel()

	got, err := client.GetCourse(ctx, &coursev1.GetCourseRequest{Id: 1})
	fmt.Println("GetCourse(1)     ->", got.GetCourse().GetTitle(), err)

	list, _ := client.ListCourses(ctx, &coursev1.ListCoursesRequest{Category: "backend"})
	fmt.Println("ListCourses      ->", len(list.GetCourses()), "of", list.GetTotal())

	_, err = client.GetCourse(ctx, &coursev1.GetCourseRequest{Id: 99})
	fmt.Println("GetCourse(99)    ->", status.Code(err), "|", status.Convert(err).Message())

	stream, _ := client.WatchEnrollments(ctx, &coursev1.WatchEnrollmentsRequest{CourseId: 1})
	for range 3 {
		ev, err := stream.Recv()
		if err == io.EOF {
			break
		}
		fmt.Println("stream event     ->", ev.GetStudent(), "enrolled,", ev.GetSeatsLeft(), "seats left")
	}
}
