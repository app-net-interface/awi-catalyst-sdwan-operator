// Copyright (c) 2023 Cisco Systems, Inc. and its affiliates
// All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http:www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log"
	"time"

	awi "github.com/app-net-interface/awi-grpc/pb"
	"google.golang.org/grpc"
)

const (
	address = "localhost:50051"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := awi.NewServiceControllerClient(conn)

	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	csr := &awi.ConnectServiceRequest{
		DestInfo: &awi.ConnectServiceRequest_DestInfo{
			Scope:    awi.Scope_PRIVATE,
			Name:     "test.viptela.com",
			Ip:       "10.10.10.10",
			Port:     "8443",
			Protocol: "tcp",
		},
		LocalWorkload: &awi.Workload{},
		SlaRequest:    &awi.SLARequest{},
	}
	cr, err := c.Connect(ctx, csr)
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", cr.GetConnectionInfo())

}
