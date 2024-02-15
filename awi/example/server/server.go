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
	"log"
	"net"
	"reflect"

	awiGrpc "github.com/app-net-interface/awi-grpc/pb"

	"google.golang.org/grpc"
)

const (
	port = ":50051"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	awiGrpc.UnimplementedServiceControllerServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) Connect(ctx context.Context, csr *awiGrpc.VirtualConnectionRequest) (*awiGrpc.VirtualConnectionResponse, error) {
	//log.Printf("Received %v request for : %v", reflect.ValueOf(s).MethodByName("Connect").Call([]reflect.Value{}), csr.DestInfo)
	log.Printf("%v", csr)
	return &awiGrpc.VirtualConnectionResponse{
		Status:         awiGrpc.Status_IN_PROGRESS,
		ConnectionId:   "12345",
		ConnectionName: "test",
		Error:          nil,
		State:          awiGrpc.State_UP,
	}, nil
}

func (s *server) Disconnect(ctx context.Context, csr *awiGrpc.DisconnectRequest) (*awiGrpc.DisconnectResponse, error) {
	//log.Printf("Received %v request for : %v", reflect.ValueOf(s).MethodByName("Connect").Call([]reflect.Value{}), csr.DestInfo)
	log.Printf("%v", csr)
	return &awiGrpc.DisconnectResponse{
		ConnectionName: "test",
		Error:          &awiGrpc.Error{},
		Status:         0,
		State:          awiGrpc.State_DOWN,
	}, nil
}

func (s *server) GetConnection(ctx context.Context, gsr *awiGrpc.GetVirtualConnectionRequest) (*awiGrpc.VirtualConnectionResponse, error) {

	log.Printf("Received %v request for : %v", reflect.ValueOf(s).MethodByName("GetConnection").Call([]reflect.Value{}), gsr.ConnectionId)

	return &awiGrpc.VirtualConnectionResponse{}, nil
}

func (s *server) ListConnections(ctx context.Context, lc *awiGrpc.ListVirtualConnectionsRequest) (*awiGrpc.ListVirtualConnectionsResponse, error) {
	log.Printf("Received %v request for : %v", reflect.ValueOf(s).MethodByName("ListConnections").Call([]reflect.Value{}), lc.String())

	return &awiGrpc.ListVirtualConnectionsResponse{}, nil
}

func (s *server) GetConnectionStatus(ctx context.Context, gcs *awiGrpc.VirtualConnectionStatusRequest) (*awiGrpc.VirtualConnectionStatusResponse, error) {
	log.Printf("Received %v request for : %v", reflect.ValueOf(s).MethodByName("GetConnectionStatus").Call([]reflect.Value{}), gcs.ConnectionId)

	return &awiGrpc.VirtualConnectionStatusResponse{}, nil
}

func (s *server) GetConnectionStatistics(ctx context.Context, gcs *awiGrpc.VirtualConnectionStatisticsRequest) (*awiGrpc.VirtualConnectionStatisticsResponse, error) {
	log.Printf("Received %v request for : %v", reflect.ValueOf(s).MethodByName("GetConnectionStatistics").Call([]reflect.Value{}), gcs.ConnectionId)

	return &awiGrpc.VirtualConnectionStatisticsResponse{}, nil
}

func (s *server) GetConnectionEvents(ctx context.Context, gce *awiGrpc.VirtualConnectionEventsRequest) (*awiGrpc.VirtualConnectionEventsResponse, error) {
	log.Printf("Received %v request for : %v", reflect.ValueOf(s).MethodByName("GetConnectionEvents").Call([]reflect.Value{}), gce.ConnectionId)

	return &awiGrpc.VirtualConnectionEventsResponse{}, nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	awiGrpc.RegisterServiceControllerServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
