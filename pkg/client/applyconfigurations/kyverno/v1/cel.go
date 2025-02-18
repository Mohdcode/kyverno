/*
Copyright The Kubernetes Authors.

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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

import (
	v1 "k8s.io/api/admissionregistration/v1"
)

// CELApplyConfiguration represents an declarative configuration of the CEL type for use
// with apply.
type CELApplyConfiguration struct {
	Expressions      []v1.Validation      `json:"expressions,omitempty"`
	ParamKind        *v1.ParamKind        `json:"paramKind,omitempty"`
	ParamRef         *v1.ParamRef         `json:"paramRef,omitempty"`
	AuditAnnotations []v1.AuditAnnotation `json:"auditAnnotations,omitempty"`
	Variables        []v1.Variable        `json:"variables,omitempty"`
}

// CELApplyConfiguration constructs an declarative configuration of the CEL type for use with
// apply.
func CEL() *CELApplyConfiguration {
	return &CELApplyConfiguration{}
}

// WithExpressions adds the given value to the Expressions field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Expressions field.
func (b *CELApplyConfiguration) WithExpressions(values ...v1.Validation) *CELApplyConfiguration {
	for i := range values {
		b.Expressions = append(b.Expressions, values[i])
	}
	return b
}

// WithParamKind sets the ParamKind field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ParamKind field is set to the value of the last call.
func (b *CELApplyConfiguration) WithParamKind(value v1.ParamKind) *CELApplyConfiguration {
	b.ParamKind = &value
	return b
}

// WithParamRef sets the ParamRef field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ParamRef field is set to the value of the last call.
func (b *CELApplyConfiguration) WithParamRef(value v1.ParamRef) *CELApplyConfiguration {
	b.ParamRef = &value
	return b
}

// WithAuditAnnotations adds the given value to the AuditAnnotations field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the AuditAnnotations field.
func (b *CELApplyConfiguration) WithAuditAnnotations(values ...v1.AuditAnnotation) *CELApplyConfiguration {
	for i := range values {
		b.AuditAnnotations = append(b.AuditAnnotations, values[i])
	}
	return b
}

// WithVariables adds the given value to the Variables field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Variables field.
func (b *CELApplyConfiguration) WithVariables(values ...v1.Variable) *CELApplyConfiguration {
	for i := range values {
		b.Variables = append(b.Variables, values[i])
	}
	return b
}
