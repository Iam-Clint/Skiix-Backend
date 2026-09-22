package infrastructure

import (
	"fmt"
	"os"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
)

// NewCloudinary initializes Cloudinary client from env vars.
// Required env:
// - CLOUDINARY_CLOUD_NAME
// - CLOUDINARY_API_KEY
// - CLOUDINARY_API_SECRET
func NewCloudinary() (*cloudinary.Cloudinary, error) {
	cloudName := strings.TrimSpace(os.Getenv("CLOUDINARY_CLOUD_NAME"))
	apiKey := strings.TrimSpace(os.Getenv("CLOUDINARY_API_KEY"))
	apiSecret := strings.TrimSpace(os.Getenv("CLOUDINARY_API_SECRET"))
	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("cloudinary env missing (CLOUDINARY_CLOUD_NAME/API_KEY/API_SECRET)")
	}
	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return nil, err
	}
	return cld, nil
}
