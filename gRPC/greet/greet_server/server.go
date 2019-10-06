package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"net"
	"strconv"
	"time"

	"../greetprotobuf"
	"google.golang.org/grpc"
)

type server struct{}

func (*server) Greet(ctx context.Context, req *greetprotobuf.GreetRequest) (*greetprotobuf.GreetResponse, error) {
	fmt.Printf("Greet function whas invoked with %v", req)
	firstName := req.GetGreeting().GetFirstName()

	result := "Hello " + firstName
	response := &greetprotobuf.GreetResponse{
		Result: result,
	}

	return response, nil
}

func (*server) GreetManyTimes(req *greetprotobuf.GreetManyTimesRequest, stream greetprotobuf.GreetService_GreetManyTimesServer) error {
	fmt.Printf("GreetManyTimes function whas invoked with %v", req)
	firstName := req.GetGreeting().GetFirstName()
	for i := 0; i < 10; i++ {
		result := "Hello " + firstName + " number " + strconv.Itoa(i)
		response := &greetprotobuf.GreetManyTimesResponse{
			Result: result,
		}
		stream.Send(response)
		time.Sleep(1000 * time.Millisecond)
	}
	return nil
}

func (*server) LongGreet(stream greetprotobuf.GreetService_LongGreetServer) error {
	fmt.Printf("LongGreet function whas invoked with a streaming request\n")
	result := ""
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// we have finished reading the client stream
			return stream.SendAndClose(&greetprotobuf.LongGreetResponse{
				Result: result,
			})
		}
		if err != nil {
			log.Fatalf("Error while reading client stream: %v", err)
		}

		firstName := req.GetGreeting().GetFirstName()
		result += "Hello " + firstName + "! "
	}
}

func (*server) GreetEveryone(stream greetprotobuf.GreetService_GreetEveryoneServer) error {
	fmt.Printf("GreetEveryone function whas invoked with a streaming request\n")

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			log.Fatalf("Error while reading client stream: %v", err)
		}
		firstName := req.GetGreeting().GetFirstName()
		result := "Hello " + firstName + "! "

		sendErr := stream.Send(&greetprotobuf.GreetEveryoneResponse{
			Result: result,
		})
		if sendErr != nil {
			log.Fatalf("Error while sending data to client stream: %v", err)
			return err
		}
	}
}

func (*server) GreetWithDeadLine(ctx context.Context, req *greetprotobuf.GreetWithDeadLineRequest) (*greetprotobuf.GreetWithDeadLineResponse, error) {
	fmt.Printf("GreetWithDeadline function whas invoked with %v", req)
	for i := 0; i < 3; i++ {
		if ctx.Err() == context.Canceled {
			fmt.Println("The client canceled the request!")
			return nil, status.Error(codes.Canceled, "The client canceled the request.")
		}
		time.Sleep(1 * time.Second)
	}
	firstName := req.GetGreeting().GetFirstName()
	result := "Hello " + firstName
	res := &greetprotobuf.GreetWithDeadLineResponse{
		Result: result,
	}
	return res, nil
}

func main() {
	fmt.Println("Greet Server!")

	listener, errors := net.Listen("tcp", "0.0.0.0:50051")
	if errors != nil {
		log.Fatalf("Failed to listen: %v", errors)
	}

	certFile := "C:/Users/shane/Desktop/SelfSignedCerts/server.crt"
	keyFile := "C:/Users/shane/Desktop/SelfSignedCerts/server.key"
	creds, sslErr := credentials.NewServerTLSFromFile(certFile, keyFile)
	if sslErr != nil {
		log.Fatalf("Failed loading certificates: %v", sslErr)
		return
	}
	options := grpc.Creds(creds)

	newServer := grpc.NewServer(options)
	greetprotobuf.RegisterGreetServiceServer(newServer, &server{})

	if errors := newServer.Serve(listener); errors != nil {
		log.Fatalf("Failed to serve: %v", errors)
	}
}
