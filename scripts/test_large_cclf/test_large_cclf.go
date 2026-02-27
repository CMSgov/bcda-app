package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/pprof"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {

	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	f, err := os.Create("./memprofile.prof")
	if err != nil {
		log.Fatal("could not create memory profile: ", err)
	}
	defer f.Close()

	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, func(lo *config.LoadOptions) error {
		lo.Region = "us-east-1"
		return nil
	})
	if err != nil {
		fmt.Printf("\naws config error: %+v\n", err)
	}
	s3Client := s3.NewFromConfig(cfg) //, func(o *s3.Options) {
	// 	o.Region = "us-east1"
	// })
	bucket := "testing-cclf-files"
	filePath := "file.zip" // zip with valid cclf files as well as large garbage file
	// filePath := "garbage.txt" // large single file

	// set up mem check
	// fmt.Printf("\nTotalAlloc before openFileBytes: %+v\n", stats)

	// get file old way
	buff, err := openFileBytes(ctx, s3Client, bucket, filePath)
	if err != nil {
		fmt.Printf("\nopenFileBytes error: %+v\n", err)
	}
	fmt.Printf("buff[0] %v", buff[0])

	// runtime.GC()

	// mem check
	// fmt.Printf("\nTotalAlloc after openFileBytes: %+v", stats)

	// get file new way
	ba, err := streamFile(ctx, s3Client, bucket, filePath)
	if err != nil {
		fmt.Printf("\nstreamFile error: %+v\n", err)
	}

	// mem check
	// fmt.Printf("\nTotalAlloc after streamFile: %+v\n", stats)

	// time.Sleep(1000 * time.Millisecond)

	// runtime.GC()

	// parse zip archive
	reader, err := zip.NewReader(bytes.NewReader(ba), int64(len(ba)))
	if err != nil {
		fmt.Printf("\nzip reader err: %+v\n", err)
	}
	for _, f := range reader.File {
		fmt.Printf("\nfile data: %+v\n", f.FileHeader)
	}

	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatal("could not write heap profile: ", err)
	}
}

func openFileBytes(ctx context.Context, s3Client *s3.Client, bucket, filePath string) ([]byte, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filePath),
	}
	output, err := s3Client.HeadObject(ctx, input)
	if err != nil {
		return nil, err
	}

	buff := make([]byte, int(*output.ContentLength))
	w := manager.NewWriteAtBuffer(buff)

	downloader := manager.NewDownloader(s3Client)
	numBytes, err := downloader.Download(ctx, w, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filePath),
	})
	if err != nil {
		return nil, err
	}

	fmt.Printf("file_size_bytes %+v\n", numBytes)

	return buff, err
}

func streamFile(ctx context.Context, s3Client *s3.Client, bucket, filePath string) ([]byte, error) {
	f, err := os.CreateTemp("/tmp", "tmp*.txt")
	if err != nil {
		return nil, err
	}

	downloader := manager.NewDownloader(s3Client)
	numBytes, err := downloader.Download(ctx, f, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filePath),
	})
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fmt.Printf("file_size_bytes %+v\n", numBytes)

	bytes, err := os.ReadFile(f.Name())
	if err != nil {
		fmt.Printf("\nreadfile err: %+v\n", err)
	}

	return bytes, nil
}
