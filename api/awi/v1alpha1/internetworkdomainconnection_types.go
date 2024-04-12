/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	awi "github.com/app-net-interface/awi-grpc/pb"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// InterNetworkDomainConnection is the Schema for the internetworkdomainConnections API
type InterNetworkDomainConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   awi.ConnectionRequest              `json:"spec,omitempty"`
	Status InterNetworkDomainConnectionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// InterNetworkDomainConnectionList contains a list of InterNetworkDomainConnection
type InterNetworkDomainConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InterNetworkDomainConnection `json:"items"`
}

type InterNetworkDomainConnectionStatus struct {
	State        string `json:"state,omitempty"`
	ConnectionId string `json:"connection_id,omitempty"`
}

func init() {
	SchemeBuilder.Register(&InterNetworkDomainConnection{}, &InterNetworkDomainConnectionList{})
}
