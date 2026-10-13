# 19: gRPC

REST/JSON is great for browsers and public APIs. Between **internal services**, many Go teams use **gRPC**:
binary Protocol Buffers over HTTP/2, a strict contract, generated clients in every language, and streaming.

```
proto/course/v1/course.proto   the contract (messages + service)
gen/course/v1/*.pb.go          GENERATED code (committed; never edit by hand)
server.go                      implements the generated interface using lesson 12's course.Service
interceptors.go                logging + auth "middleware"
cmd/server, cmd/client         runnable server and client
```

## Concepts
- **Protocol Buffers**: typed messages with **field numbers**. Adding fields is backward compatible.
  Never renumber or reuse numbers. `buf breaking` detects breaking changes.
- **Code generation** with `buf` (`buf.gen.yaml`) + `protoc-gen-go` + `protoc-gen-go-grpc`. `buf lint` enforces
  good API style (it made us give every RPC its own request/response message).
- **RPC kinds**: unary (`GetCourse`) and **server streaming** (`WatchEnrollments`). Client streaming and
  bidirectional streaming also exist.
- **Status codes** (`codes.NotFound`, `InvalidArgument`, `AlreadyExists`, `Unauthenticated`...) instead of HTTP statuses.
  Domain errors are mapped in one place (`toStatus`).
- **Metadata** (gRPC headers) carries auth tokens. **Interceptors** are middleware: logging, auth, metrics.
- **Deadlines**: every RPC should have one (`context.WithTimeout`). They propagate to the server, where `ctx.Done()` stops streams.
- `UnimplementedCourseServiceServer` embedding gives forward compatibility.
- **Health service** and **reflection** (for `grpcurl`), `GracefulStop`.
- **Testing** with `bufconn`: a real gRPC server and client over an in-memory connection.
- Same service, two transports: lesson 12 (REST) and this lesson (gRPC) share `course.Service`.
  **Connect** (connectrpc.com) and **grpc-gateway** can serve both from one definition.

## Run
```bash
go run ./19-grpc/cmd/server
go run ./19-grpc/cmd/client
grpcurl -plaintext -H 'authorization: Bearer demo-token' -d '{"id":1}' localhost:9090 course.v1.CourseService/GetCourse
go test -v ./19-grpc

# After editing the .proto (install buf, protoc-gen-go and protoc-gen-go-grpc first):
cd 19-grpc && buf lint && buf generate
```
