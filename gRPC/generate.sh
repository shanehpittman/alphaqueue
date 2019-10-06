#!/bin/bash

#Run command from the /gRPC folder in terminal.
.\protoc ./queue/queuepb/queue.proto --go_out=plugins=grpc:.

# go run .\gRPC\queue\queue_server\server.go
# go run .\gRPC\queue\queue_server\server.go
