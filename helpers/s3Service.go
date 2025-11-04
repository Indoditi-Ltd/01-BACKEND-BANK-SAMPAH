package helpers

import (
	"backend-mulungs/configs"
	"context"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Service struct {
	client   *s3.Client
	bucket   string
	endpoint string
}

func NewS3Service() *S3Service {
	return &S3Service{
		client:   configs.S3Client,
		bucket:   configs.GetS3Bucket(),
		endpoint: configs.GetAWSEndpoint(),
	}
}

// GetBucket - Getter untuk bucket name
func (s *S3Service) GetBucket() string {
	return s.bucket
}

// GetEndpoint - Getter untuk endpoint
func (s *S3Service) GetEndpoint() string {
	return s.endpoint
}

// UploadFile mengupload file ke NEO Object Storage
func (s *S3Service) UploadFile(file *multipart.FileHeader, userID uint, folder string) (string, error) {
	// Buka file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Generate unique filename
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		return "", fmt.Errorf("file has no extension")
	}

	uniqueName := fmt.Sprintf("%s/user_%d_%d%s", folder, userID, time.Now().Unix(), ext)
	fmt.Printf("Uploading file. Bucket: %s, Key: %s\n", s.bucket, uniqueName)

	// Cek apakah client S3 terinisialisasi
	if s.client == nil {
		return "", fmt.Errorf("S3 client is not initialized")
	}

	// Upload ke NEO Object Storage
	uploader := manager.NewUploader(s.client)
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(uniqueName),
		Body:        src,
		ContentType: aws.String(s.getContentType(ext)),
		ACL:         "public-read",
	})

	if err != nil {
		return "", fmt.Errorf("NEO Object Storage upload failed: %w", err)
	}

	fileURL := s.GetFileURL(uniqueName)
	fmt.Printf("File uploaded successfully: %s\n", fileURL)

	return fileURL, nil
}

// GetFileURL mendapatkan URL file dari NEO Object Storage
// Format: https://nos.jkt-1.neo.id/bucket-name/folder/file.jpg
func (s *S3Service) GetFileURL(key string) string {
	if key == "" {
		return ""
	}
	
	// Untuk NEO Object Storage - Format: https://endpoint/bucket/key
	if s.endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", s.endpoint, s.bucket, key)
	}
	
	// Fallback untuk AWS S3 standard
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", 
		s.bucket, 
		configs.GetAWSRegion(), 
		key)
}

// DeleteFile menghapus file dari NEO Object Storage
func (s *S3Service) DeleteFile(key string) error {
	_, err := s.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

// Extract key dari URL NEO Object Storage
// Input: https://nos.jkt-1.neo.id/image-mulungs/admin-profiles/user_1_1234567890.jpg
// Output: admin-profiles/user_1_1234567890.jpg
func (s *S3Service) ExtractKeyFromURL(url string) string {
	if s.endpoint != "" {
		prefix := fmt.Sprintf("%s/%s/", s.endpoint, s.bucket)
		if strings.HasPrefix(url, prefix) {
			return strings.TrimPrefix(url, prefix)
		}
	}
	
	// Fallback: coba extract dengan cara lain
	parts := strings.Split(url, "/")
	if len(parts) > 4 {
		// Asumsi format: https://nos.jkt-1.neo.id/bucket/key1/key2
		return strings.Join(parts[4:], "/")
	}
	
	return ""
}

// TestConnection - Test koneksi ke NEO Object Storage
func (s *S3Service) TestConnection() error {
	// Coba list objects (lebih ringan dari HeadBucket)
	_, err := s.client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket:  aws.String(s.bucket),
		MaxKeys: aws.Int32(1),
	})
	
	return err
}

// Helper function untuk content type
func (s *S3Service) getContentType(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}