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

// PURPOSE: Implements a mutating webhook that rewrites container image registries based on namespace configuration
package v1

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"mutating-registry-hook/internal/registry"
)

// nolint:unused
// log is for logging in this package.
var podlog = logf.Log.WithName("pod-resource")

const (
	LabelRegistryRewrite     = "registry-rewrite"
	LabelValueEnabled        = "enabled"
	AnnotationTargetRegistry = "image-rewriter.example.com/target-registry"
)

// SetupPodWebhookWithManager registers the webhook for Pod in the manager.
func SetupPodWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&corev1.Pod{}).
		WithValidator(&PodCustomValidator{}).
		WithDefaulter(&PodCustomDefaulter{
			Client: mgr.GetClient(),
		}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate--v1-pod,mutating=true,failurePolicy=Ignore,sideEffects=None,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod-v1.kb.io,admissionReviewVersions=v1

// PodCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Pod when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type PodCustomDefaulter struct {
	Client client.Client
}

var _ webhook.CustomDefaulter = &PodCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Pod.
func (d *PodCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	pod, ok := obj.(*corev1.Pod)

	if !ok {
		return fmt.Errorf("expected an Pod object but got %T", obj)
	}
	podlog.Info("Defaulting for Pod", "name", pod.GetName())

	// Get the namespace
	namespace := &corev1.Namespace{}
	if err := d.Client.Get(ctx, types.NamespacedName{Name: pod.Namespace}, namespace); err != nil {
		podlog.Error(err, "failed to get namespace", "namespace", pod.Namespace)
		return nil // fail-safe: don't block pod creation
	}

	// Check for label
	if namespace.Labels[LabelRegistryRewrite] != LabelValueEnabled {
		return nil // not enabled for this namespace
	}

	// Check for annotation
	targetRegistry, ok := namespace.Annotations[AnnotationTargetRegistry]
	if !ok || targetRegistry == "" {
		podlog.Info("skipping pod - missing target registry annotation", "namespace", pod.Namespace)
		return nil
	}

	// Rewrite images for all container types
	for i := range pod.Spec.Containers {
		rewritten, err := registry.RewriteImage(pod.Spec.Containers[i].Image, targetRegistry)
		if err != nil {
			podlog.Error(err, "failed to rewrite image", "original", pod.Spec.Containers[i].Image)
			continue // fail-safe: skip this container
		}
		pod.Spec.Containers[i].Image = rewritten
	}

	// Rewrite init container images
	for i := range pod.Spec.InitContainers {
		rewritten, err := registry.RewriteImage(pod.Spec.InitContainers[i].Image, targetRegistry)
		if err != nil {
			podlog.Error(err, "failed to rewrite init container image", "original", pod.Spec.InitContainers[i].Image)
			continue // fail-safe: skip this container
		}
		pod.Spec.InitContainers[i].Image = rewritten
	}

	// Rewrite ephemeral container images
	for i := range pod.Spec.EphemeralContainers {
		rewritten, err := registry.RewriteImage(pod.Spec.EphemeralContainers[i].Image, targetRegistry)
		if err != nil {
			podlog.Error(err, "failed to rewrite ephemeral container image", "original", pod.Spec.EphemeralContainers[i].Image)
			continue // fail-safe: skip this container
		}
		pod.Spec.EphemeralContainers[i].Image = rewritten
	}

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate--v1-pod,mutating=false,failurePolicy=Ignore,sideEffects=None,groups="",resources=pods,verbs=create;update,versions=v1,name=vpod-v1.kb.io,admissionReviewVersions=v1

// PodCustomValidator struct is responsible for validating the Pod resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type PodCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &PodCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Pod.
func (v *PodCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("expected a Pod object but got %T", obj)
	}
	podlog.Info("Validation for Pod upon creation", "name", pod.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Pod.
func (v *PodCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	pod, ok := newObj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("expected a Pod object for the newObj but got %T", newObj)
	}
	podlog.Info("Validation for Pod upon update", "name", pod.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Pod.
func (v *PodCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("expected a Pod object but got %T", obj)
	}
	podlog.Info("Validation for Pod upon deletion", "name", pod.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
