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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JSMTeamSpec defines the desired state of JSMTeam.
type JSMTeamSpec struct {
	// Human-readable name of the team
	Name string `json:"name"`

	// Optional: ARI of the team if known
	ID string `json:"id,omitempty"`
}

// JSMTeamStatus defines the observed state of JSMTeam.
type JSMTeamStatus struct {
	// The resolved or confirmed team ARI
	ID                 string `json:"id,omitempty"`
	ObservedGeneration int64  `json:"observedGeneration,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// JSMTeam is the Schema for the jsmteams API.
type JSMTeam struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JSMTeamSpec   `json:"spec,omitempty"`
	Status JSMTeamStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// JSMTeamList contains a list of JSMTeam.
type JSMTeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JSMTeam `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JSMTeam{}, &JSMTeamList{})
}
