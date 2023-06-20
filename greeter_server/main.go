/*
 *
 * Copyright 2015 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Package main implements a server for Greeter service.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	pb "myApp"

	"github.com/joho/godotenv"
	"github.com/sashabaranov/go-openai"
	"google.golang.org/grpc"
)

var (
	port   = flag.Int("port", 50051, "The server port")
	client *openai.Client
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
}

func (s *server) GetAnswer(ctx context.Context, in *pb.GetAnswerRequest) (*pb.GetAnswerResponse, error) {
	req := protoToInternalGetAnswerRequest(in)
	resp, err := sendToChatGPT(req, client)
	if err != nil {
		return nil, errors.New("failed making your text Shakespearean")
	}
	proto := internalGetAnswerResponseToProto(resp)
	return proto, nil
}

// mapper functions and types
type internalGetAnswerRequest struct {
	Question string
}

type internalGetAnswerResponse struct {
	Answer string
}

// protoToInternalGetAnswerRequest converts between *pb.GetAnswerRequest to internalGetAnswerRequest
func protoToInternalGetAnswerRequest(proto *pb.GetAnswerRequest) *internalGetAnswerRequest {
	return &internalGetAnswerRequest{
		Question: proto.Question,
	}
}

// internalGetAnswerRequestToProto converts between internalGetAnswerRequest to *pb.GetAnswerRequest
func internalGetAnswerRequestToProto(in *internalGetAnswerRequest) *pb.GetAnswerRequest {
	return &pb.GetAnswerRequest{
		Question: in.Question,
	}
}

// protoToInternalGetAnswerResponse converts between *pb.GetAnswerResponse to internalGetAnswerResponse
func protoToInternalGetAnswerResponse(proto *pb.GetAnswerResponse) *internalGetAnswerResponse {
	return &internalGetAnswerResponse{
		Answer: proto.Answer,
	}
}

// internalGetAnswerResponseToProto converts between internalGetAnswerResponse to *pb.GetAnswerResponse
func internalGetAnswerResponseToProto(in *internalGetAnswerResponse) *pb.GetAnswerResponse {
	return &pb.GetAnswerResponse{
		Answer: in.Answer,
	}
}

func (req *internalGetAnswerRequest) enrich() {
	req.Question = "rewrite " + req.Question + "in the voice of Shakespeare"
}

// setup package
func setupEnv() *openai.Client {
	if os.Getenv("environment") != "live" {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("could not load the .env!")
		}
	}
	GPT_API_KEY := os.Getenv("GPT_API_KEY")
	if GPT_API_KEY == "" {
		log.Fatal("GPT_API_KEY not set!")
	}
	return openai.NewClient(GPT_API_KEY)
}

//chatGPT call package

func sendToChatGPT(req *internalGetAnswerRequest, client *openai.Client) (resp2 *internalGetAnswerResponse, err error) {
	//enrich our users prompt
	req.enrich()

	//send it off to chatGPT
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: req.Question,
				},
			},
		},
	)
	//make sure there is no error when getting something back from chatgpt
	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return nil, err
	}

	//send our response proto back
	return &internalGetAnswerResponse{Answer: resp.Choices[0].Message.Content}, nil
}

func main() {
	flag.Parse()
	client = setupEnv()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
