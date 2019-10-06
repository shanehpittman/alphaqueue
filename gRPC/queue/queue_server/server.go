package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"gopkg.in/mgo.v2/bson"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"../queuepb"
	"google.golang.org/grpc"
)

var collection *mongo.Collection

type server struct {
}

type queueItem struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	AuthorID string             `bson:"author_id"`
	Content  string             `bson:"content"`
	Title    string             `bson:"title"`
}

func main() {
	// if we crash the go code, we get the file name and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	fmt.Println("Connecting to MongoDB")
	// connect to MongoDB
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	err = client.Connect(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Queue Service Started")
	collection = client.Database("mydb").Collection("queue")

	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	var opts []grpc.ServerOption
	newServer := grpc.NewServer(opts...)
	queuepb.RegisterQueueServiceServer(newServer, &server{})
	// Register reflection service on gRPC server.
	reflection.Register(newServer)

	go func() {
		fmt.Println("Starting Server...")
		if err := newServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Wait for Control C to exit
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt)

	// Block until a signal is received
	<-channel
	fmt.Println("Stopping the server")
	newServer.Stop()
	fmt.Println("Closing the listener")
	lis.Close()
	fmt.Println("Closing MongoDB Connection")
	client.Disconnect(context.TODO())
	fmt.Println("End of Program")
}

func (*server) CreateQueue(ctx context.Context, req *queuepb.CreateQueueRequest) (*queuepb.CreateQueueResponse, error) {
	fmt.Println("Create queue request")
	queue := req.GetQueue()

	data := queueItem{
		AuthorID: queue.GetAuthorId(),
		Title:    queue.GetTitle(),
		Content:  queue.GetContent(),
	}

	res, err := collection.InsertOne(context.Background(), data)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %v", err),
		)
	}

	oid, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot convert to OID"),
		)
	}

	return &queuepb.CreateQueueResponse{
		Queue: &queuepb.Queue{
			Id:       oid.Hex(),
			AuthorId: queue.GetAuthorId(),
			Title:    queue.GetTitle(),
			Content:  queue.GetContent(),
		},
	}, nil
}

func (*server) ReadQueue(ctx context.Context, req *queuepb.ReadQueueRequest) (*queuepb.ReadQueueResponse, error) {
	fmt.Println("Read queue request")

	queueID := req.GetQueueId()
	oid, err := primitive.ObjectIDFromHex(queueID)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Cannot parse ID"),
		)
	}

	// create an empty struct
	data := &queueItem{}
	filter := bson.M{"_id": oid}

	res := collection.FindOne(context.Background(), filter)
	if err := res.Decode(data); err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find queue with specified ID: %v", err),
		)
	}

	return &queuepb.ReadQueueResponse{
		Queue: dataToQueuePb(data),
	}, nil
}

func dataToQueuePb(data *queueItem) *queuepb.Queue {
	return &queuepb.Queue{
		Id:       data.ID.Hex(),
		AuthorId: data.AuthorID,
		Content:  data.Content,
		Title:    data.Title,
	}
}

func (*server) UpdateQueue(ctx context.Context, req *queuepb.UpdateQueueRequest) (*queuepb.UpdateQueueResponse, error) {
	fmt.Println("Update queue request")
	queue := req.GetQueue()
	oid, err := primitive.ObjectIDFromHex(queue.GetId())
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Cannot parse ID"),
		)
	}

	// create an empty struct
	data := &queueItem{}
	filter := bson.M{"_id": oid}

	res := collection.FindOne(context.Background(), filter)
	if err := res.Decode(data); err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find queue with specified ID: %v", err),
		)
	}

	// we update our internal struct
	data.AuthorID = queue.GetAuthorId()
	data.Content = queue.GetContent()
	data.Title = queue.GetTitle()

	_, updateErr := collection.ReplaceOne(context.Background(), filter, data)
	if updateErr != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot update object in MongoDB: %v", updateErr),
		)
	}

	return &queuepb.UpdateQueueResponse{
		Queue: dataToQueuePb(data),
	}, nil

}

func (*server) DeleteQueue(ctx context.Context, req *queuepb.DeleteQueueRequest) (*queuepb.DeleteQueueResponse, error) {
	fmt.Println("Delete queue request")
	oid, err := primitive.ObjectIDFromHex(req.GetQueueId())
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			fmt.Sprintf("Cannot parse ID"),
		)
	}

	filter := bson.M{"_id": oid}

	res, err := collection.DeleteOne(context.Background(), filter)

	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot delete object in MongoDB: %v", err),
		)
	}

	if res.DeletedCount == 0 {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find queue in MongoDB: %v", err),
		)
	}

	return &queuepb.DeleteQueueResponse{QueueId: req.GetQueueId()}, nil
}

func (*server) ListQueue(req *queuepb.ListQueueRequest, stream queuepb.QueueService_ListQueueServer) error {
	fmt.Println("List queue request")

	cur, err := collection.Find(context.Background(), primitive.D{{}})
	if err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown internal error: %v", err),
		)
	}
	defer cur.Close(context.Background())
	for cur.Next(context.Background()) {
		data := &queueItem{}
		err := cur.Decode(data)
		if err != nil {
			return status.Errorf(
				codes.Internal,
				fmt.Sprintf("Error while decoding data from MongoDB: %v", err),
			)

		}
		stream.Send(&queuepb.ListQueueResponse{Queue: dataToQueuePb(data)})
	}
	if err := cur.Err(); err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown internal error: %v", err),
		)
	}
	return nil
}
