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

// JSMServiceSpec defines the desired state of JSMService.
type JSMServiceSpec struct {
	// Human-readable name of the service
	Name string `json:"name,omitempty"`

	// Optional service description
	Description string `json:"description,omitempty"`

	// Service tier level (1-4), required for creation
	TierLevel int `json:"tierLevel"`

	// Optional: service type key (e.g., APPLICATIONS, BUSINESS_SERVICES)
	ServiceTypeKey string `json:"serviceTypeKey,omitempty"`

	// Reference to a JSMTeam for responders
	TeamRef *JSMTeamRef `json:"teamRef,omitempty"`
}

// JSMTeamRef allows referencing a JSMTeam object
type JSMTeamRef struct {
	// Name of the JSMTeam resource
	Name string `json:"name"`
}

// JSMServiceStatus defines the observed state of JSMService.
type JSMServiceStatus struct {
	// Standard Kubernetes status conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Custom fields (e.g., ID, Revision, etc.)
	ID                 string `json:"id,omitempty"`
	Revision           string `json:"revision,omitempty"`
	ObservedGeneration int64  `json:"observedGeneration,omitempty"`
	TierID             string `json:"tierID,omitempty"`
	TierLevel          int    `json:"tierLevel,omitempty"`
	TeamRelationshipID string `json:"teamRelationshipID,omitempty"`
	ResolvedTeamARN    string `json:"resolvedTeamARN,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// JSMService is the Schema for the jsmservices API.
type JSMService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JSMServiceSpec   `json:"spec,omitempty"`
	Status JSMServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// JSMServiceList contains a list of JSMService.
type JSMServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JSMService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&JSMService{}, &JSMServiceList{})
}
