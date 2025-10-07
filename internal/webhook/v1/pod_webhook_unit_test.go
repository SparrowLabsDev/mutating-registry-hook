// PURPOSE: Unit tests for Pod webhook defaulter using fake clients
package v1

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testNginxImage      = "nginx:latest"
	testMyRegistryNginx = "myregistry.io/nginx:latest"
)

func TestPodDefaulter_SkipNamespacesWithoutLabel(t *testing.T) {
	// Create a namespace without the label
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
		},
	}

	// Create a pod in that namespace
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: testNginxImage,
				},
			},
		},
	}

	// Create a fake client with the namespace
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(namespace).Build()

	// Create defaulter with the client
	defaulter := PodCustomDefaulter{
		Client: fakeClient,
	}

	// Call Default method
	err := defaulter.Default(context.Background(), pod)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify image was not modified
	if pod.Spec.Containers[0].Image != testNginxImage {
		t.Errorf("Expected image to remain testNginxImage, got: %s", pod.Spec.Containers[0].Image)
	}
}

func TestPodDefaulter_SkipWhenAnnotationMissing(t *testing.T) {
	// Create a namespace with the label but without the annotation
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
			Labels: map[string]string{
				LabelRegistryRewrite: LabelValueEnabled,
			},
		},
	}

	// Create a pod in that namespace
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: testNginxImage,
				},
			},
		},
	}

	// Create a fake client with the namespace
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(namespace).Build()

	// Create defaulter with the client
	defaulter := PodCustomDefaulter{
		Client: fakeClient,
	}

	// Call Default method
	err := defaulter.Default(context.Background(), pod)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify image was not modified
	if pod.Spec.Containers[0].Image != testNginxImage {
		t.Errorf("Expected image to remain testNginxImage, got: %s", pod.Spec.Containers[0].Image)
	}
}

func TestPodDefaulter_RewriteRegularContainers(t *testing.T) {
	// Create a namespace with both label and annotation
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
			Labels: map[string]string{
				LabelRegistryRewrite: LabelValueEnabled,
			},
			Annotations: map[string]string{
				AnnotationTargetRegistry: "myregistry.io",
			},
		},
	}

	// Create a pod with multiple containers
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "docker.io/nginx:latest",
				},
				{
					Name:  "redis",
					Image: "redis:7.0",
				},
			},
		},
	}

	// Create a fake client with the namespace
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(namespace).Build()

	// Create defaulter with the client
	defaulter := PodCustomDefaulter{
		Client: fakeClient,
	}

	// Call Default method
	err := defaulter.Default(context.Background(), pod)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify images were rewritten
	expectedImages := []string{
		testMyRegistryNginx,
		"myregistry.io/redis:7.0",
	}

	for i, container := range pod.Spec.Containers {
		if container.Image != expectedImages[i] {
			t.Errorf("Container %d: expected image '%s', got '%s'", i, expectedImages[i], container.Image)
		}
	}
}

func TestPodDefaulter_RewriteInitContainers(t *testing.T) {
	// Create a namespace with both label and annotation
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
			Labels: map[string]string{
				LabelRegistryRewrite: LabelValueEnabled,
			},
			Annotations: map[string]string{
				AnnotationTargetRegistry: "myregistry.io",
			},
		},
	}

	// Create a pod with init containers
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
		},
		Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{
				{
					Name:  "init-db",
					Image: "postgres:15",
				},
				{
					Name:  "init-config",
					Image: "busybox:1.36",
				},
			},
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: testNginxImage,
				},
			},
		},
	}

	// Create a fake client with the namespace
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(namespace).Build()

	// Create defaulter with the client
	defaulter := PodCustomDefaulter{
		Client: fakeClient,
	}

	// Call Default method
	err := defaulter.Default(context.Background(), pod)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify init container images were rewritten
	expectedInitImages := []string{
		"myregistry.io/postgres:15",
		"myregistry.io/busybox:1.36",
	}

	for i, container := range pod.Spec.InitContainers {
		if container.Image != expectedInitImages[i] {
			t.Errorf("InitContainer %d: expected image '%s', got '%s'", i, expectedInitImages[i], container.Image)
		}
	}

	// Verify regular container image was also rewritten
	if pod.Spec.Containers[0].Image != testMyRegistryNginx {
		t.Errorf("Container: expected image testMyRegistryNginx, got '%s'", pod.Spec.Containers[0].Image)
	}
}

func TestPodDefaulter_RewriteEphemeralContainers(t *testing.T) {
	// Create a namespace with both label and annotation
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
			Labels: map[string]string{
				LabelRegistryRewrite: LabelValueEnabled,
			},
			Annotations: map[string]string{
				AnnotationTargetRegistry: "myregistry.io",
			},
		},
	}

	// Create a pod with ephemeral containers
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "app",
					Image: testNginxImage,
				},
			},
			EphemeralContainers: []corev1.EphemeralContainer{
				{
					EphemeralContainerCommon: corev1.EphemeralContainerCommon{
						Name:  "debug",
						Image: "busybox:1.36",
					},
				},
			},
		},
	}

	// Create a fake client with the namespace
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(namespace).Build()

	// Create defaulter with the client
	defaulter := PodCustomDefaulter{
		Client: fakeClient,
	}

	// Call Default method
	err := defaulter.Default(context.Background(), pod)

	// Verify no error occurred
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Verify ephemeral container image was rewritten
	if pod.Spec.EphemeralContainers[0].Image != "myregistry.io/busybox:1.36" {
		t.Errorf("EphemeralContainer: expected image 'myregistry.io/busybox:1.36', got '%s'", pod.Spec.EphemeralContainers[0].Image)
	}

	// Verify regular container image was also rewritten
	if pod.Spec.Containers[0].Image != testMyRegistryNginx {
		t.Errorf("Container: expected image testMyRegistryNginx, got '%s'", pod.Spec.Containers[0].Image)
	}
}

func TestPodDefaulter_HandlesErrorsGracefully(t *testing.T) {
	// Create a namespace with both label and annotation
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
			Labels: map[string]string{
				LabelRegistryRewrite: LabelValueEnabled,
			},
			Annotations: map[string]string{
				AnnotationTargetRegistry: "myregistry.io",
			},
		},
	}

	// Create a pod with an empty image (which will cause rewriter to error)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "good-container",
					Image: testNginxImage,
				},
				{
					Name:  "bad-container",
					Image: "", // This will cause an error
				},
				{
					Name:  "another-good-container",
					Image: "redis:7.0",
				},
			},
		},
	}

	// Create a fake client with the namespace
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(namespace).Build()

	// Create defaulter with the client
	defaulter := PodCustomDefaulter{
		Client: fakeClient,
	}

	// Call Default method
	err := defaulter.Default(context.Background(), pod)

	// Verify no error was returned (fail-safe behavior)
	if err != nil {
		t.Errorf("Expected no error to be returned (fail-safe), got: %v", err)
	}

	// Verify good containers were rewritten
	if pod.Spec.Containers[0].Image != testMyRegistryNginx {
		t.Errorf("Container 0: expected image testMyRegistryNginx, got '%s'", pod.Spec.Containers[0].Image)
	}

	// Verify bad container was skipped (image remains empty)
	if pod.Spec.Containers[1].Image != "" {
		t.Errorf("Container 1: expected image to remain empty (skipped due to error), got '%s'", pod.Spec.Containers[1].Image)
	}

	// Verify another good container was rewritten
	if pod.Spec.Containers[2].Image != "myregistry.io/redis:7.0" {
		t.Errorf("Container 2: expected image 'myregistry.io/redis:7.0', got '%s'", pod.Spec.Containers[2].Image)
	}
}

func TestPodDefaulter_NamespaceNotFound(t *testing.T) {
	// Create a pod in a namespace that doesn't exist
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "nonexistent-namespace",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: testNginxImage,
				},
			},
		},
	}

	// Create a fake client WITHOUT the namespace
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// Create defaulter with the client
	defaulter := PodCustomDefaulter{
		Client: fakeClient,
	}

	// Call Default method
	err := defaulter.Default(context.Background(), pod)

	// Verify no error was returned (fail-safe behavior)
	if err != nil {
		t.Errorf("Expected no error to be returned (fail-safe), got: %v", err)
	}

	// Verify image was not modified
	if pod.Spec.Containers[0].Image != testNginxImage {
		t.Errorf("Expected image to remain testNginxImage, got: %s", pod.Spec.Containers[0].Image)
	}
}
