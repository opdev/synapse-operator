//go:build !ignore_autogenerated
// +build !ignore_autogenerated

/*
Copyright 2021.

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
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Synapse) DeepCopyInto(out *Synapse) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Synapse.
func (in *Synapse) DeepCopy() *Synapse {
	if in == nil {
		return nil
	}
	out := new(Synapse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Synapse) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SynapseBridges) DeepCopyInto(out *SynapseBridges) {
	*out = *in
	out.Heisenbridge = in.Heisenbridge
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SynapseBridges.
func (in *SynapseBridges) DeepCopy() *SynapseBridges {
	if in == nil {
		return nil
	}
	out := new(SynapseBridges)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SynapseHeisenbridge) DeepCopyInto(out *SynapseHeisenbridge) {
	*out = *in
	out.ConfigMap = in.ConfigMap
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SynapseHeisenbridge.
func (in *SynapseHeisenbridge) DeepCopy() *SynapseHeisenbridge {
	if in == nil {
		return nil
	}
	out := new(SynapseHeisenbridge)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SynapseHeisenbridgeConfigMap) DeepCopyInto(out *SynapseHeisenbridgeConfigMap) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SynapseHeisenbridgeConfigMap.
func (in *SynapseHeisenbridgeConfigMap) DeepCopy() *SynapseHeisenbridgeConfigMap {
	if in == nil {
		return nil
	}
	out := new(SynapseHeisenbridgeConfigMap)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SynapseHomeserver) DeepCopyInto(out *SynapseHomeserver) {
	*out = *in
	if in.ConfigMap != nil {
		in, out := &in.ConfigMap, &out.ConfigMap
		*out = new(SynapseHomeserverConfigMap)
		**out = **in
	}
	if in.Values != nil {
		in, out := &in.Values, &out.Values
		*out = new(SynapseHomeserverValues)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SynapseHomeserver.
func (in *SynapseHomeserver) DeepCopy() *SynapseHomeserver {
	if in == nil {
		return nil
	}
	out := new(SynapseHomeserver)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SynapseHomeserverConfigMap) DeepCopyInto(out *SynapseHomeserverConfigMap) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SynapseHomeserverConfigMap.
func (in *SynapseHomeserverConfigMap) DeepCopy() *SynapseHomeserverConfigMap {
	if in == nil {
		return nil
	}
	out := new(SynapseHomeserverConfigMap)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SynapseHomeserverValues) DeepCopyInto(out *SynapseHomeserverValues) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SynapseHomeserverValues.
func (in *SynapseHomeserverValues) DeepCopy() *SynapseHomeserverValues {
	if in == nil {
		return nil
	}
	out := new(SynapseHomeserverValues)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SynapseList) DeepCopyInto(out *SynapseList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Synapse, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SynapseList.
func (in *SynapseList) DeepCopy() *SynapseList {
	if in == nil {
		return nil
	}
	out := new(SynapseList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *SynapseList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SynapseSpec) DeepCopyInto(out *SynapseSpec) {
	*out = *in
	in.Homeserver.DeepCopyInto(&out.Homeserver)
	out.Bridges = in.Bridges
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SynapseSpec.
func (in *SynapseSpec) DeepCopy() *SynapseSpec {
	if in == nil {
		return nil
	}
	out := new(SynapseSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SynapseStatus) DeepCopyInto(out *SynapseStatus) {
	*out = *in
	out.BridgesConfiguration = in.BridgesConfiguration
	out.DatabaseConnectionInfo = in.DatabaseConnectionInfo
	out.HomeserverConfiguration = in.HomeserverConfiguration
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SynapseStatus.
func (in *SynapseStatus) DeepCopy() *SynapseStatus {
	if in == nil {
		return nil
	}
	out := new(SynapseStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SynapseStatusBridgesConfiguration) DeepCopyInto(out *SynapseStatusBridgesConfiguration) {
	*out = *in
	out.Heisenbridge = in.Heisenbridge
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SynapseStatusBridgesConfiguration.
func (in *SynapseStatusBridgesConfiguration) DeepCopy() *SynapseStatusBridgesConfiguration {
	if in == nil {
		return nil
	}
	out := new(SynapseStatusBridgesConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SynapseStatusDatabaseConnectionInfo) DeepCopyInto(out *SynapseStatusDatabaseConnectionInfo) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SynapseStatusDatabaseConnectionInfo.
func (in *SynapseStatusDatabaseConnectionInfo) DeepCopy() *SynapseStatusDatabaseConnectionInfo {
	if in == nil {
		return nil
	}
	out := new(SynapseStatusDatabaseConnectionInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SynapseStatusHeisenbridge) DeepCopyInto(out *SynapseStatusHeisenbridge) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SynapseStatusHeisenbridge.
func (in *SynapseStatusHeisenbridge) DeepCopy() *SynapseStatusHeisenbridge {
	if in == nil {
		return nil
	}
	out := new(SynapseStatusHeisenbridge)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SynapseStatusHomeserverConfiguration) DeepCopyInto(out *SynapseStatusHomeserverConfiguration) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SynapseStatusHomeserverConfiguration.
func (in *SynapseStatusHomeserverConfiguration) DeepCopy() *SynapseStatusHomeserverConfiguration {
	if in == nil {
		return nil
	}
	out := new(SynapseStatusHomeserverConfiguration)
	in.DeepCopyInto(out)
	return out
}
