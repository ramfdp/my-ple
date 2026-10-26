package controller

import (
	"image"
	"testing"
)

func TestValidAvatarConfigRejectsExcessiveDimensions(t *testing.T) {
	if validAvatarConfig(image.Config{Width: 10_000, Height: 10_000}) {
		t.Fatal("oversized image dimensions accepted")
	}
	if !validAvatarConfig(image.Config{Width: 1024, Height: 1024}) {
		t.Fatal("normal avatar dimensions rejected")
	}
}
