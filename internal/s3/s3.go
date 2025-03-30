package s3

import (
	"bytes"
	"fmt"
	"log"
	"mime/multipart"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var (
	s3Client *s3.S3
	bucket   = "emerge-data-bucket" // update if needed
)

func Init() {
	sess := session.Must(session.NewSession())
	s3Client = s3.New(sess)
}

func UploadFile(file multipart.File, fileHeader *multipart.FileHeader, folder string) (string, error) {
	defer file.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(file); err != nil {
		return "", err
	}

	key := fmt.Sprintf("%s/%d_%s", folder, time.Now().UnixNano(), fileHeader.Filename)
	contentType := fileHeader.Header.Get("Content-Type")

	_, err := s3Client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String(contentType),
		ACL:         aws.String("public-read"),
	})
	if err != nil {
		log.Printf("❌ S3 upload failed: %v", err)
		return "", err
	}

	url := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", bucket, key)
	return url, nil
}
