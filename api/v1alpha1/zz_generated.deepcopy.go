//go:build !ignore_autogenerated

/*
Copyright 2024.

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsCredentialsKeysRef) DeepCopyInto(out *AwsCredentialsKeysRef) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsCredentialsKeysRef.
func (in *AwsCredentialsKeysRef) DeepCopy() *AwsCredentialsKeysRef {
	if in == nil {
		return nil
	}
	out := new(AwsCredentialsKeysRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsCredentialsSpec) DeepCopyInto(out *AwsCredentialsSpec) {
	*out = *in
	if in.KeysRef != nil {
		in, out := &in.KeysRef, &out.KeysRef
		*out = new(AwsCredentialsKeysRef)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsCredentialsSpec.
func (in *AwsCredentialsSpec) DeepCopy() *AwsCredentialsSpec {
	if in == nil {
		return nil
	}
	out := new(AwsCredentialsSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsDocumentDBCluster) DeepCopyInto(out *AwsDocumentDBCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.StatusType.DeepCopyInto(&out.StatusType)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsDocumentDBCluster.
func (in *AwsDocumentDBCluster) DeepCopy() *AwsDocumentDBCluster {
	if in == nil {
		return nil
	}
	out := new(AwsDocumentDBCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AwsDocumentDBCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsDocumentDBClusterAction) DeepCopyInto(out *AwsDocumentDBClusterAction) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsDocumentDBClusterAction.
func (in *AwsDocumentDBClusterAction) DeepCopy() *AwsDocumentDBClusterAction {
	if in == nil {
		return nil
	}
	out := new(AwsDocumentDBClusterAction)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsDocumentDBClusterIdentifier) DeepCopyInto(out *AwsDocumentDBClusterIdentifier) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsDocumentDBClusterIdentifier.
func (in *AwsDocumentDBClusterIdentifier) DeepCopy() *AwsDocumentDBClusterIdentifier {
	if in == nil {
		return nil
	}
	out := new(AwsDocumentDBClusterIdentifier)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsDocumentDBClusterList) DeepCopyInto(out *AwsDocumentDBClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AwsDocumentDBCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsDocumentDBClusterList.
func (in *AwsDocumentDBClusterList) DeepCopy() *AwsDocumentDBClusterList {
	if in == nil {
		return nil
	}
	out := new(AwsDocumentDBClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AwsDocumentDBClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsDocumentDBClusterSpec) DeepCopyInto(out *AwsDocumentDBClusterSpec) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	if in.ServiceEndpoint != nil {
		in, out := &in.ServiceEndpoint, &out.ServiceEndpoint
		*out = new(string)
		**out = **in
	}
	if in.DBClusterIdentifiers != nil {
		in, out := &in.DBClusterIdentifiers, &out.DBClusterIdentifiers
		*out = make([]AwsDocumentDBClusterIdentifier, len(*in))
		copy(*out, *in)
	}
	if in.Actions != nil {
		in, out := &in.Actions, &out.Actions
		*out = make(map[string]AwsDocumentDBClusterAction, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.Credentials != nil {
		in, out := &in.Credentials, &out.Credentials
		*out = new(AwsCredentialsSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsDocumentDBClusterSpec.
func (in *AwsDocumentDBClusterSpec) DeepCopy() *AwsDocumentDBClusterSpec {
	if in == nil {
		return nil
	}
	out := new(AwsDocumentDBClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsEc2Instance) DeepCopyInto(out *AwsEc2Instance) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.StatusType.DeepCopyInto(&out.StatusType)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsEc2Instance.
func (in *AwsEc2Instance) DeepCopy() *AwsEc2Instance {
	if in == nil {
		return nil
	}
	out := new(AwsEc2Instance)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AwsEc2Instance) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsEc2InstanceAction) DeepCopyInto(out *AwsEc2InstanceAction) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	if in.Force != nil {
		in, out := &in.Force, &out.Force
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsEc2InstanceAction.
func (in *AwsEc2InstanceAction) DeepCopy() *AwsEc2InstanceAction {
	if in == nil {
		return nil
	}
	out := new(AwsEc2InstanceAction)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsEc2InstanceIdentifier) DeepCopyInto(out *AwsEc2InstanceIdentifier) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsEc2InstanceIdentifier.
func (in *AwsEc2InstanceIdentifier) DeepCopy() *AwsEc2InstanceIdentifier {
	if in == nil {
		return nil
	}
	out := new(AwsEc2InstanceIdentifier)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsEc2InstanceList) DeepCopyInto(out *AwsEc2InstanceList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AwsEc2Instance, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsEc2InstanceList.
func (in *AwsEc2InstanceList) DeepCopy() *AwsEc2InstanceList {
	if in == nil {
		return nil
	}
	out := new(AwsEc2InstanceList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AwsEc2InstanceList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsEc2InstanceSpec) DeepCopyInto(out *AwsEc2InstanceSpec) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	if in.ServiceEndpoint != nil {
		in, out := &in.ServiceEndpoint, &out.ServiceEndpoint
		*out = new(string)
		**out = **in
	}
	if in.Instances != nil {
		in, out := &in.Instances, &out.Instances
		*out = make([]AwsEc2InstanceIdentifier, len(*in))
		copy(*out, *in)
	}
	if in.Actions != nil {
		in, out := &in.Actions, &out.Actions
		*out = make(map[string]AwsEc2InstanceAction, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.Credentials != nil {
		in, out := &in.Credentials, &out.Credentials
		*out = new(AwsCredentialsSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsEc2InstanceSpec.
func (in *AwsEc2InstanceSpec) DeepCopy() *AwsEc2InstanceSpec {
	if in == nil {
		return nil
	}
	out := new(AwsEc2InstanceSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsRdsAuroraCluster) DeepCopyInto(out *AwsRdsAuroraCluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.StatusType.DeepCopyInto(&out.StatusType)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsRdsAuroraCluster.
func (in *AwsRdsAuroraCluster) DeepCopy() *AwsRdsAuroraCluster {
	if in == nil {
		return nil
	}
	out := new(AwsRdsAuroraCluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AwsRdsAuroraCluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsRdsAuroraClusterAction) DeepCopyInto(out *AwsRdsAuroraClusterAction) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsRdsAuroraClusterAction.
func (in *AwsRdsAuroraClusterAction) DeepCopy() *AwsRdsAuroraClusterAction {
	if in == nil {
		return nil
	}
	out := new(AwsRdsAuroraClusterAction)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsRdsAuroraClusterIdentifier) DeepCopyInto(out *AwsRdsAuroraClusterIdentifier) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsRdsAuroraClusterIdentifier.
func (in *AwsRdsAuroraClusterIdentifier) DeepCopy() *AwsRdsAuroraClusterIdentifier {
	if in == nil {
		return nil
	}
	out := new(AwsRdsAuroraClusterIdentifier)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsRdsAuroraClusterList) DeepCopyInto(out *AwsRdsAuroraClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AwsRdsAuroraCluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsRdsAuroraClusterList.
func (in *AwsRdsAuroraClusterList) DeepCopy() *AwsRdsAuroraClusterList {
	if in == nil {
		return nil
	}
	out := new(AwsRdsAuroraClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AwsRdsAuroraClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsRdsAuroraClusterSpec) DeepCopyInto(out *AwsRdsAuroraClusterSpec) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	if in.ServiceEndpoint != nil {
		in, out := &in.ServiceEndpoint, &out.ServiceEndpoint
		*out = new(string)
		**out = **in
	}
	if in.DBClusterIdentifiers != nil {
		in, out := &in.DBClusterIdentifiers, &out.DBClusterIdentifiers
		*out = make([]AwsRdsAuroraClusterIdentifier, len(*in))
		copy(*out, *in)
	}
	if in.Actions != nil {
		in, out := &in.Actions, &out.Actions
		*out = make(map[string]AwsRdsAuroraClusterAction, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.Credentials != nil {
		in, out := &in.Credentials, &out.Credentials
		*out = new(AwsCredentialsSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsRdsAuroraClusterSpec.
func (in *AwsRdsAuroraClusterSpec) DeepCopy() *AwsRdsAuroraClusterSpec {
	if in == nil {
		return nil
	}
	out := new(AwsRdsAuroraClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsTransferFamily) DeepCopyInto(out *AwsTransferFamily) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.StatusType.DeepCopyInto(&out.StatusType)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsTransferFamily.
func (in *AwsTransferFamily) DeepCopy() *AwsTransferFamily {
	if in == nil {
		return nil
	}
	out := new(AwsTransferFamily)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AwsTransferFamily) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsTransferFamilyAction) DeepCopyInto(out *AwsTransferFamilyAction) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsTransferFamilyAction.
func (in *AwsTransferFamilyAction) DeepCopy() *AwsTransferFamilyAction {
	if in == nil {
		return nil
	}
	out := new(AwsTransferFamilyAction)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsTransferFamilyIdentifier) DeepCopyInto(out *AwsTransferFamilyIdentifier) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsTransferFamilyIdentifier.
func (in *AwsTransferFamilyIdentifier) DeepCopy() *AwsTransferFamilyIdentifier {
	if in == nil {
		return nil
	}
	out := new(AwsTransferFamilyIdentifier)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsTransferFamilyList) DeepCopyInto(out *AwsTransferFamilyList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]AwsTransferFamily, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsTransferFamilyList.
func (in *AwsTransferFamilyList) DeepCopy() *AwsTransferFamilyList {
	if in == nil {
		return nil
	}
	out := new(AwsTransferFamilyList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *AwsTransferFamilyList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AwsTransferFamilySpec) DeepCopyInto(out *AwsTransferFamilySpec) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	if in.ServiceEndpoint != nil {
		in, out := &in.ServiceEndpoint, &out.ServiceEndpoint
		*out = new(string)
		**out = **in
	}
	if in.ServerIds != nil {
		in, out := &in.ServerIds, &out.ServerIds
		*out = make([]AwsTransferFamilyIdentifier, len(*in))
		copy(*out, *in)
	}
	if in.Actions != nil {
		in, out := &in.Actions, &out.Actions
		*out = make(map[string]AwsTransferFamilyAction, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.Credentials != nil {
		in, out := &in.Credentials, &out.Credentials
		*out = new(AwsCredentialsSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AwsTransferFamilySpec.
func (in *AwsTransferFamilySpec) DeepCopy() *AwsTransferFamilySpec {
	if in == nil {
		return nil
	}
	out := new(AwsTransferFamilySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sHpa) DeepCopyInto(out *K8sHpa) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.StatusType.DeepCopyInto(&out.StatusType)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sHpa.
func (in *K8sHpa) DeepCopy() *K8sHpa {
	if in == nil {
		return nil
	}
	out := new(K8sHpa)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *K8sHpa) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sHpaAction) DeepCopyInto(out *K8sHpaAction) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sHpaAction.
func (in *K8sHpaAction) DeepCopy() *K8sHpaAction {
	if in == nil {
		return nil
	}
	out := new(K8sHpaAction)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sHpaList) DeepCopyInto(out *K8sHpaList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]K8sHpa, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sHpaList.
func (in *K8sHpaList) DeepCopy() *K8sHpaList {
	if in == nil {
		return nil
	}
	out := new(K8sHpaList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *K8sHpaList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sHpaSpec) DeepCopyInto(out *K8sHpaSpec) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	if in.Namespaces != nil {
		in, out := &in.Namespaces, &out.Namespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.LabelSelector.DeepCopyInto(&out.LabelSelector)
	if in.Actions != nil {
		in, out := &in.Actions, &out.Actions
		*out = make(map[string]K8sHpaAction, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sHpaSpec.
func (in *K8sHpaSpec) DeepCopy() *K8sHpaSpec {
	if in == nil {
		return nil
	}
	out := new(K8sHpaSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sPodReplicas) DeepCopyInto(out *K8sPodReplicas) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.StatusType.DeepCopyInto(&out.StatusType)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sPodReplicas.
func (in *K8sPodReplicas) DeepCopy() *K8sPodReplicas {
	if in == nil {
		return nil
	}
	out := new(K8sPodReplicas)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *K8sPodReplicas) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sPodReplicasAction) DeepCopyInto(out *K8sPodReplicasAction) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sPodReplicasAction.
func (in *K8sPodReplicasAction) DeepCopy() *K8sPodReplicasAction {
	if in == nil {
		return nil
	}
	out := new(K8sPodReplicasAction)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sPodReplicasList) DeepCopyInto(out *K8sPodReplicasList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]K8sPodReplicas, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sPodReplicasList.
func (in *K8sPodReplicasList) DeepCopy() *K8sPodReplicasList {
	if in == nil {
		return nil
	}
	out := new(K8sPodReplicasList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *K8sPodReplicasList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sPodReplicasSpec) DeepCopyInto(out *K8sPodReplicasSpec) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	if in.Namespaces != nil {
		in, out := &in.Namespaces, &out.Namespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.LabelSelector.DeepCopyInto(&out.LabelSelector)
	if in.Actions != nil {
		in, out := &in.Actions, &out.Actions
		*out = make(map[string]K8sPodReplicasAction, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sPodReplicasSpec.
func (in *K8sPodReplicasSpec) DeepCopy() *K8sPodReplicasSpec {
	if in == nil {
		return nil
	}
	out := new(K8sPodReplicasSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sRunJob) DeepCopyInto(out *K8sRunJob) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.StatusType.DeepCopyInto(&out.StatusType)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sRunJob.
func (in *K8sRunJob) DeepCopy() *K8sRunJob {
	if in == nil {
		return nil
	}
	out := new(K8sRunJob)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *K8sRunJob) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sRunJobAction) DeepCopyInto(out *K8sRunJobAction) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	in.Job.DeepCopyInto(&out.Job)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sRunJobAction.
func (in *K8sRunJobAction) DeepCopy() *K8sRunJobAction {
	if in == nil {
		return nil
	}
	out := new(K8sRunJobAction)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sRunJobList) DeepCopyInto(out *K8sRunJobList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]K8sRunJob, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sRunJobList.
func (in *K8sRunJobList) DeepCopy() *K8sRunJobList {
	if in == nil {
		return nil
	}
	out := new(K8sRunJobList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *K8sRunJobList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *K8sRunJobSpec) DeepCopyInto(out *K8sRunJobSpec) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	if in.Namespaces != nil {
		in, out := &in.Namespaces, &out.Namespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Actions != nil {
		in, out := &in.Actions, &out.Actions
		*out = make(map[string]K8sRunJobAction, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new K8sRunJobSpec.
func (in *K8sRunJobSpec) DeepCopy() *K8sRunJobSpec {
	if in == nil {
		return nil
	}
	out := new(K8sRunJobSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NotificationPolicy) DeepCopyInto(out *NotificationPolicy) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.StatusType.DeepCopyInto(&out.StatusType)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NotificationPolicy.
func (in *NotificationPolicy) DeepCopy() *NotificationPolicy {
	if in == nil {
		return nil
	}
	out := new(NotificationPolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NotificationPolicy) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NotificationPolicyApiSpec) DeepCopyInto(out *NotificationPolicyApiSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NotificationPolicyApiSpec.
func (in *NotificationPolicyApiSpec) DeepCopy() *NotificationPolicyApiSpec {
	if in == nil {
		return nil
	}
	out := new(NotificationPolicyApiSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NotificationPolicyList) DeepCopyInto(out *NotificationPolicyList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]NotificationPolicy, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NotificationPolicyList.
func (in *NotificationPolicyList) DeepCopy() *NotificationPolicyList {
	if in == nil {
		return nil
	}
	out := new(NotificationPolicyList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NotificationPolicyList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NotificationPolicyMSTeamsSpec) DeepCopyInto(out *NotificationPolicyMSTeamsSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NotificationPolicyMSTeamsSpec.
func (in *NotificationPolicyMSTeamsSpec) DeepCopy() *NotificationPolicyMSTeamsSpec {
	if in == nil {
		return nil
	}
	out := new(NotificationPolicyMSTeamsSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NotificationPolicySlackSpec) DeepCopyInto(out *NotificationPolicySlackSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NotificationPolicySlackSpec.
func (in *NotificationPolicySlackSpec) DeepCopy() *NotificationPolicySlackSpec {
	if in == nil {
		return nil
	}
	out := new(NotificationPolicySlackSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NotificationPolicySpec) DeepCopyInto(out *NotificationPolicySpec) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	if in.Schedules != nil {
		in, out := &in.Schedules, &out.Schedules
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.Api != nil {
		in, out := &in.Api, &out.Api
		*out = new(NotificationPolicyApiSpec)
		**out = **in
	}
	if in.MSTeams != nil {
		in, out := &in.MSTeams, &out.MSTeams
		*out = new(NotificationPolicyMSTeamsSpec)
		**out = **in
	}
	if in.Slack != nil {
		in, out := &in.Slack, &out.Slack
		*out = new(NotificationPolicySlackSpec)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NotificationPolicySpec.
func (in *NotificationPolicySpec) DeepCopy() *NotificationPolicySpec {
	if in == nil {
		return nil
	}
	out := new(NotificationPolicySpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Schedule) DeepCopyInto(out *Schedule) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.StatusType.DeepCopyInto(&out.StatusType)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Schedule.
func (in *Schedule) DeepCopy() *Schedule {
	if in == nil {
		return nil
	}
	out := new(Schedule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Schedule) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScheduleAction) DeepCopyInto(out *ScheduleAction) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScheduleAction.
func (in *ScheduleAction) DeepCopy() *ScheduleAction {
	if in == nil {
		return nil
	}
	out := new(ScheduleAction)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScheduleList) DeepCopyInto(out *ScheduleList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Schedule, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScheduleList.
func (in *ScheduleList) DeepCopy() *ScheduleList {
	if in == nil {
		return nil
	}
	out := new(ScheduleList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ScheduleList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScheduleSpec) DeepCopyInto(out *ScheduleSpec) {
	*out = *in
	if in.Enabled != nil {
		in, out := &in.Enabled, &out.Enabled
		*out = new(bool)
		**out = **in
	}
	if in.Actions != nil {
		in, out := &in.Actions, &out.Actions
		*out = make(map[string]ScheduleAction, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.ActivePeriods != nil {
		in, out := &in.ActivePeriods, &out.ActivePeriods
		*out = make([]TimePeriod, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.InactivePeriods != nil {
		in, out := &in.InactivePeriods, &out.InactivePeriods
		*out = make([]TimePeriod, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScheduleSpec.
func (in *ScheduleSpec) DeepCopy() *ScheduleSpec {
	if in == nil {
		return nil
	}
	out := new(ScheduleSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Status) DeepCopyInto(out *Status) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Status.
func (in *Status) DeepCopy() *Status {
	if in == nil {
		return nil
	}
	out := new(Status)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StatusType) DeepCopyInto(out *StatusType) {
	*out = *in
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StatusType.
func (in *StatusType) DeepCopy() *StatusType {
	if in == nil {
		return nil
	}
	out := new(StatusType)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TimePeriod) DeepCopyInto(out *TimePeriod) {
	*out = *in
	in.Start.DeepCopyInto(&out.Start)
	in.End.DeepCopyInto(&out.End)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TimePeriod.
func (in *TimePeriod) DeepCopy() *TimePeriod {
	if in == nil {
		return nil
	}
	out := new(TimePeriod)
	in.DeepCopyInto(out)
	return out
}
