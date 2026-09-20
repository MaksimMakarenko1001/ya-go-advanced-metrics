## protobuf

## HOWTO
```bash
protoc \
  --go_out=. --go_opt=paths=source_relative \
  --go_opt=Mapi/proto/metrics/metrics.proto=github.com/MaksimMakarenko1001/ya-go-advanced/api/proto/metrics \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  --go-grpc_opt=Mapi/proto/metrics/metrics.proto=github.com/MaksimMakarenko1001/ya-go-advanced/api/proto/metrics \
  --go_opt=default_api_level=API_OPAQUE \
  api/proto/metrics/metrics.proto
```