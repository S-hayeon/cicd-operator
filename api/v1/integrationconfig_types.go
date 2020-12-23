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
	"context"
	"fmt"
	tektonv1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tmax-cloud/cicd-operator/internal/configs"

	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IntegrationConfigSpec defines the desired state of IntegrationConfig
type IntegrationConfigSpec struct {
	// Git config for target repository
	Git GitConfig `json:"git"`

	// Secrets are the list of secret names which are included in service account
	Secrets []corev1.LocalObjectReference `json:"secrets,omitempty"`

	// Workspaces list
	Workspaces []tektonv1beta1.WorkspaceBinding `json:"workspaces,omitempty"`

	// Jobs specify the tasks to be executed
	Jobs IntegrationConfigJobs `json:"jobs"`

	// Merge
	// TODO
}

type IntegrationConfigJobs struct {
	// PreSubmit jobs are for pull-request events
	PreSubmit []Job `json:"preSubmit,omitempty"`

	// PostSubmit jobs are for push events (including tag events)
	PostSubmit []Job `json:"postSubmit,omitempty"`
}

// IntegrationConfigStatus defines the observed state of IntegrationConfig
type IntegrationConfigStatus struct {
	// Conditions of IntegrationConfig
	Conditions status.Conditions `json:"conditions"`
	Secrets    string            `json:"secrets"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// IntegrationConfig is the Schema for the integrationconfigs API
// +kubebuilder:resource:shortName="ic"
type IntegrationConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IntegrationConfigSpec   `json:"spec,omitempty"`
	Status IntegrationConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// IntegrationConfigList contains a list of IntegrationConfig
type IntegrationConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IntegrationConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IntegrationConfig{}, &IntegrationConfigList{})
}

func (i *IntegrationConfig) GetToken(c client.Client) (string, error) {
	tokenStruct := i.Spec.Git.Token
	// Get from value
	if tokenStruct.ValueFrom == nil {
		if tokenStruct.Value != "" {
			return tokenStruct.Value, nil
		} else {
			return "", fmt.Errorf("token is empty")
		}
	}

	// Get from secret
	secretName := tokenStruct.ValueFrom.SecretKeyRef.Name
	secretKey := tokenStruct.ValueFrom.SecretKeyRef.Key
	secret := &corev1.Secret{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: i.Namespace}, secret); err != nil {
		return "", err
	}
	token, ok := secret.Data[secretKey]
	if !ok {
		return "", fmt.Errorf("token secret/key %s/%s not valid", secretName, secretKey)
	}
	return string(token), nil
}

func GetServiceAccountName(configName string) string {
	return fmt.Sprintf("%s-sa", configName)
}

func GetSecretName(configName string) string {
	return configName
}

// Returns Server address which webhook events will be received
func (i *IntegrationConfig) GetWebhookServerAddress() string {
	return fmt.Sprintf("http://%s/webhook/%s/%s", configs.ExternalHostName, i.Namespace, i.Name)
}

const (
	IntegrationConfigConditionWebhookRegistered = status.ConditionType("webhook-registered")
	IntegrationConfigConditionReady             = status.ConditionType("ready")
)
