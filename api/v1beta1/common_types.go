package v1beta1

type VarKeySelector struct {
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:Required
	Key string `json:"key,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.configMapKeyRef) || has(self.secretKeyRef)",message="One of configMapKeyRef and secretKeyRef must be set"
// +kubebuilder:validation:XValidation:rule="!has(self.configMapKeyRef) || !has(self.secretKeyRef)",message="Only one of configMapKeyRef and secretKeyRef can be set"
type VarSource struct {
	ConfigMapKeyRef *VarKeySelector `json:"configMapKeyRef,omitempty"`
	SecretKeyRef    *VarKeySelector `json:"secretKeyRef,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.value) || has(self.valueFrom)",message="One of value and valueFrom must be set"
// +kubebuilder:validation:XValidation:rule="!has(self.value) || !has(self.valueFrom)",message="Only one of value and valueFrom can be set"
type VarValue struct {
	Value     string     `json:"value,omitempty"`
	ValueFrom *VarSource `json:"valueFrom,omitempty"`
}
