// PURPOSE: Test suite for container image reference rewriting functionality
package registry

import (
	"testing"
)

const (
	testTargetRegistry = "target-registry.com"
	testExpectedImage  = "target-registry.com/nginx:latest"
)

func TestRewriteImage_SimpleImageWithTag(t *testing.T) {
	originalImage := "nginx:latest"
	targetRegistry := testTargetRegistry
	expected := testExpectedImage

	result, err := RewriteImage(originalImage, targetRegistry)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != expected {
		t.Errorf("RewriteImage(%q, %q) = %q; want %q", originalImage, targetRegistry, result, expected)
	}
}

func TestRewriteImage_SimpleImageWithoutTag(t *testing.T) {
	originalImage := "nginx"
	targetRegistry := testTargetRegistry
	expected := testExpectedImage

	result, err := RewriteImage(originalImage, targetRegistry)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != expected {
		t.Errorf("RewriteImage(%q, %q) = %q; want %q", originalImage, targetRegistry, result, expected)
	}
}

func TestRewriteImage_DockerHubRegistry(t *testing.T) {
	originalImage := "docker.io/library/nginx:latest"
	targetRegistry := testTargetRegistry
	expected := "target-registry.com/library/nginx:latest"

	result, err := RewriteImage(originalImage, targetRegistry)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != expected {
		t.Errorf("RewriteImage(%q, %q) = %q; want %q", originalImage, targetRegistry, result, expected)
	}
}

func TestRewriteImage_GCRWithPath(t *testing.T) {
	originalImage := "gcr.io/project/app:v1"
	targetRegistry := testTargetRegistry
	expected := "target-registry.com/project/app:v1"

	result, err := RewriteImage(originalImage, targetRegistry)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != expected {
		t.Errorf("RewriteImage(%q, %q) = %q; want %q", originalImage, targetRegistry, result, expected)
	}
}

func TestRewriteImage_WithDigest(t *testing.T) {
	originalImage := "nginx@sha256:abc123def456"
	targetRegistry := testTargetRegistry
	expected := "target-registry.com/nginx@sha256:abc123def456"

	result, err := RewriteImage(originalImage, targetRegistry)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != expected {
		t.Errorf("RewriteImage(%q, %q) = %q; want %q", originalImage, targetRegistry, result, expected)
	}
}

func TestRewriteImage_RegistryWithPort(t *testing.T) {
	originalImage := "localhost:5000/myapp:v1"
	targetRegistry := testTargetRegistry
	expected := "target-registry.com/myapp:v1"

	result, err := RewriteImage(originalImage, targetRegistry)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != expected {
		t.Errorf("RewriteImage(%q, %q) = %q; want %q", originalImage, targetRegistry, result, expected)
	}
}

func TestRewriteImage_DeepPath(t *testing.T) {
	originalImage := "gcr.io/project/subproject/app:v1"
	targetRegistry := testTargetRegistry
	expected := "target-registry.com/project/subproject/app:v1"

	result, err := RewriteImage(originalImage, targetRegistry)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != expected {
		t.Errorf("RewriteImage(%q, %q) = %q; want %q", originalImage, targetRegistry, result, expected)
	}
}

func TestRewriteImage_EmptyImageError(t *testing.T) {
	originalImage := ""
	targetRegistry := testTargetRegistry

	_, err := RewriteImage(originalImage, targetRegistry)

	if err == nil {
		t.Error("RewriteImage with empty image should return an error")
	}
}

func TestRewriteImage_EmptyRegistryError(t *testing.T) {
	originalImage := "nginx:latest"
	targetRegistry := ""

	_, err := RewriteImage(originalImage, targetRegistry)

	if err == nil {
		t.Error("RewriteImage with empty target registry should return an error")
	}
}

func TestRewriteImage_TargetRegistryAlreadyMatches(t *testing.T) {
	originalImage := testExpectedImage
	targetRegistry := testTargetRegistry
	expected := testExpectedImage

	result, err := RewriteImage(originalImage, targetRegistry)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != expected {
		t.Errorf("RewriteImage(%q, %q) = %q; want %q", originalImage, targetRegistry, result, expected)
	}
}
