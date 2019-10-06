package main

import (
	"context"
	"fmt"
	"io"
	"log"

	"../queuepb"
	"google.golang.org/grpc"
)

func main() {

	fmt.Println("Queue Client")

	opts := grpc.WithInsecure()

	connection, err := grpc.Dial("localhost:50051", opts)
	if err != nil {
		log.Fatalf("could not connect: %v", err)
	}
	defer connection.Close()

	client := queuepb.NewQueueServiceClient(connection)

	// create Queue
	fmt.Println("Creating the queue")
	queue := &queuepb.Queue{
		AuthorId: "Shane",
		Title:    "My First Queue",
		Content:  "Content of the first queue",
	}

	createQueueRes, err := client.CreateQueue(context.Background(), &queuepb.CreateQueueRequest{Queue: queue})
	if err != nil {
		log.Fatalf("Unexpected error: %v", err)
	}
	fmt.Printf("Queue has been created: %v", createQueueRes)
	queueID := createQueueRes.GetQueue().GetId()

	// read Queue
	fmt.Println("Reading the queue")
	_, forcedErr := client.ReadQueue(context.Background(), &queuepb.ReadQueueRequest{QueueId: "8675309"})
	if forcedErr != nil {
		fmt.Printf("Error happened while reading: %v \n", forcedErr)
	}
	readQueueReq := &queuepb.ReadQueueRequest{QueueId: queueID}
	readQueueRes, readQueueErr := client.ReadQueue(context.Background(), readQueueReq)
	if readQueueErr != nil {
		fmt.Printf("Error happened while reading: %v \n", readQueueErr)
	}
	fmt.Printf("Queue was read: %v \n", readQueueRes)

	// update Queue
	newQueue := &queuepb.Queue{
		Id:       queueID,
		AuthorId: "Changed Author",
		Title:    "My First Queue (edited)",
		Content:  "Content of the first queue, with some awesome additions!",
	}
	updateRes, updateErr := client.UpdateQueue(context.Background(), &queuepb.UpdateQueueRequest{Queue: newQueue})
	if updateErr != nil {
		fmt.Printf("Error happened while updating: %v \n", updateErr)
	}
	fmt.Printf("Queue was updated: %v\n", updateRes)

	// delete Queue
	deleteRes, deleteErr := client.DeleteQueue(context.Background(), &queuepb.DeleteQueueRequest{QueueId: queueID})

	if deleteErr != nil {
		fmt.Printf("Error happened while deleting: %v \n", deleteErr)
	}
	fmt.Printf("Queue was deleted: %v \n", deleteRes)

	// list Queues
	stream, err := client.ListQueue(context.Background(), &queuepb.ListQueueRequest{})
	if err != nil {
		log.Fatalf("error while calling ListQueue RPC: %v", err)
	}
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Something happened: %v", err)
		}
		fmt.Println(res.GetQueue())
	}
}
