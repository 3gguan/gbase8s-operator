/*


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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type Gbase8sConfigMap struct {
	Name        string `json:"name,omitempty"`
	OnconfigKey string `json:"onconfigKey,omitempty"`
	AllowedKey  string `json:"allowedKey,omitempty"`
}

type CmConfigMap struct {
	Name      string `json:"name,omitempty"`
	ConfigKey string `json:"configKey,omitempty"`
}

type Gbase8sStorage struct {
	Path       string `json:"path,omitempty"`
	Size       string `json:"size,omitempty"`
	VolumeMode string `json:"volumeMode,omitempty"`
}

type Failover struct {
	DetectingCount    int `json:"detectingCount,omitempty"`
	DetectingInterval int `json:"detectingInterval,omitempty"`
	Timeout           int `json:"timeout,omitempty"`
}

type Gbase8sNode struct {
	Name    string          `json:"name,omitempty"`
	Storage *Gbase8sStorage `json:"storage,omitempty"`
	Log     *Gbase8sStorage `json:"log,omitempty"`
}

type CmNode struct {
	Name string          `json:"name,omitempty"`
	Log  *Gbase8sStorage `json:"log,omitempty"`
}

type Gbase8sConfig struct {
	Replicas   int32            `json:"replicas,omitempty"`
	Image      string           `json:"image,omitempty"`
	Failover   Failover         `json:"failover,omitempty"`
	SecretName string           `json:"secretName,omitempty"`
	ConfigMap  Gbase8sConfigMap `json:"configMap,omitempty"`
	Nodes      []Gbase8sNode    `json:"nodes,omitempty"`
}

type CmConfig struct {
	Replicas          int32       `json:"replicas,omitempty"`
	Image             string      `json:"image,omitempty"`
	SecretName        string      `json:"secretName,omitempty"`
	RedirectGroupName string      `json:"redirectGroupName,omitempty"`
	ProxyGroupName    string      `json:"proxyGroupName,omitempty"`
	ConfigMap         CmConfigMap `json:"configMap,omitempty"`
	Nodes             []CmNode    `json:"nodes,omitempty"`
}

// Gbase8sClusterSpec defines the desired state of Gbase8sCluster
type Gbase8sClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Gbase8sCluster. Edit Gbase8sCluster_types.go to remove/update
	// Foo string `json:"foo,omitempty"`
	//Namespace  string               `json:"namespace,omitempty"`
	Gbase8sCfg Gbase8sConfig `json:"gbase8s,omitempty"`
	CmCfg      CmConfig      `json:"connectManager,omitempty"`
}

// Gbase8sClusterStatus defines the observed state of Gbase8sCluster
type Gbase8sClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status string `json:"status"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Gbase8sCluster is the Schema for the gbase8sclusters API
type Gbase8sCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Gbase8sClusterSpec   `json:"spec,omitempty"`
	Status Gbase8sClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// Gbase8sClusterList contains a list of Gbase8sCluster
type Gbase8sClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Gbase8sCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Gbase8sCluster{}, &Gbase8sClusterList{})
}
