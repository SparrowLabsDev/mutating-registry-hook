// PURPOSE: Provides functionality to parse and rewrite container image references to use a target registry
package registry

import (
	"errors"
	"fmt"
	"strings"
)

// RewriteImage takes an original container image reference and rewrites it to use the target registry
func RewriteImage(originalImage string, targetRegistry string) (string, error) {
	// Validate inputs
	if originalImage == "" {
		return "", errors.New("original image cannot be empty")
	}
	if targetRegistry == "" {
		return "", errors.New("target registry cannot be empty")
	}

	image := originalImage

	// Strip existing registry if present
	// A registry is present if there's a "/" and the part before it contains a "." or ":"
	if strings.Contains(image, "/") {
		parts := strings.SplitN(image, "/", 2)
		if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") {
			// First part is a registry, strip it
			image = parts[1]
		}
	}

	// If the image doesn't have a tag or digest, add :latest
	if !strings.Contains(image, ":") && !strings.Contains(image, "@") {
		image = image + ":latest"
	}

	return fmt.Sprintf("%s/%s", targetRegistry, image), nil
}
