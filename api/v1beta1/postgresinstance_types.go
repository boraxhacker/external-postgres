/*
Copyright 2025.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PostgresInstanceSpec defines the desired state of PostgresInstance.
type PostgresInstanceSpec struct {
	Host          VarValue `json:"host"`
	Port          VarValue `json:"port"`
	AdminUserName VarValue `json:"adminUserName"`
	AdminPassword VarValue `json:"adminPassword"`
}

// PostgresInstanceStatus defines the observed state of PostgresInstance.
type PostgresInstanceStatus struct {
	//
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// PostgresInstance is the Schema for the postgresinstances API.
type PostgresInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgresInstanceSpec   `json:"spec,omitempty"`
	Status PostgresInstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PostgresInstanceList contains a list of PostgresInstance.
type PostgresInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgresInstance{}, &PostgresInstanceList{})
}
