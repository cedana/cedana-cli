package catalog

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	cedana "github.com/cedana/cedana-cli/types"
)

var r2CatalogBucket string = "pub-e47371c3835348ea9fac25dba76439e0"
var paperspaceRegion map[string]string = map[string]string{
	"CA1":  "West Coast (CA1)",
	"NY2":  "East Coast (NY2)",
	"AMS1": "Europe (AMS1)",
}

// should be called as part of the build process!
func ParseAWSCatalog() {
	// Open the CSV file
	file, err := os.Open("market/catalog/aws.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Read the entire CSV file into memory
	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}

	data := make([]cedana.Instance, 0, len(rows)-1)
	for _, row := range rows[1:] {
		// HACK - ignore metals
		if strings.Contains(row[0], "metal") {
			continue
		}
		instance := cedana.Instance{}
		instance.Provider = "aws"
		instance.InstanceType = row[0]
		instance.AcceleratorName = row[1]
		instance.AcceleratorCount, _ = strconv.Atoi(row[2])
		instance.VCPUs, _ = strconv.ParseFloat(row[3], 64)
		instance.MemoryGiB, _ = strconv.ParseFloat(row[4], 64)
		if row[5] != "" {
			// TODO: this is wrong - json is incorrectly parsed
			gpuinfoJSON := strings.ReplaceAll(row[5], "'", "\"")
			instance.GPUs = gpuinfoJSON
		}
		instance.Region = row[8]
		instance.AvailabilityZone = row[9]
		data = append(data, instance)
	}
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		panic(err)
	}

	// Write the JSON data to a file. This is called only in main, so path is relative to main.go
	err = os.WriteFile("market/catalog/aws_catalog.json", jsonData, 0644)
	if err != nil {
		panic(err)
	}
}

func ParsePaperspaceCatalog() {

	// not having access to json file is messy - will need a separate cloud service
	// until then, we pull the json from some cloud host
	file, err := os.Open("market/catalog/paperspace.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Read the entire CSV file into memory
	reader := csv.NewReader(file)
	rows, err := reader.ReadAll()
	if err != nil {
		panic(err)
	}

	var data []cedana.Instance
	for _, row := range rows[1:] {
		priceStr := strings.Trim(row[4], "$/hr")
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			panic(err)
		}

		regions := strings.Split(row[5], ", ")

		for _, region := range regions {
			var gpuInfo string
			// if the vRAM is not empty, then we have a GPU
			if row[1] != "" {
				gpu := cedana.GpuInfo{
					Gpus: []cedana.GPU{
						{
							Name:         row[0],
							Manufacturer: "NVIDIA", // paperspace not using anything but right now anayway
							Count:        1,        // fake - not multi-GPU machines
						},
					},
					TotalGpuMemoryInMiB: gbToMb(parseInt(row[1])),
				}
				gpuJSON, _ := json.Marshal(gpu)
				gpuInfo = string(gpuJSON)
			}
			instance := cedana.Instance{
				Provider:         "paperspace",
				InstanceType:     row[0],
				AcceleratorName:  row[0],
				AcceleratorCount: 0, // need to derive this
				VCPUs:            parseFloat(row[2]),
				MemoryGiB:        parseFloat(row[3]),
				GPUs:             gpuInfo,
				Region:           paperspaceRegion[region],
				Price:            price,
			}
			data = append(data, instance)
		}
	}
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		panic(err)
	}

	// Write the JSON data to a file. This is called only in main, so path is relative to main.go
	err = os.WriteFile("market/catalog/paperspace_catalog.json", jsonData, 0644)
	if err != nil {
		panic(err)
	}
}

func UploadToR2(pth string) {
	// Hack - we want to upload the generated/parsed catalogues to R2.
	// We ideally want a better way to do this, but this should be good enough for now and will avoid nasty things like
	// embedding.

	file, err := os.Open(pth)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	client := MakeS3Client()
	_, err = client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket:      aws.String("cedana-catalog"),
		Key:         aws.String(pth),
		Body:        file,
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		panic(err)
	}
}

func DownloadFromR2(provider string) []cedana.Instance {
	var catalog []cedana.Instance
	// Hack - we want to upload the generated/parsed catalogues to R2.
	// We ideally want a better way to do this, but this should be good enough for now and will avoid nasty things like
	// embedding.

	filename := strings.Join([]string{"market/catalog/", provider + "_catalog", ".json"}, "")

	// want to hit this with just a GET request, ensuring portability
	url := fmt.Sprintf("https://%s.r2.dev/%s", r2CatalogBucket, filename)
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(body, &catalog)
	if err != nil {
		panic(err)
	}

	return catalog
}

func MakeS3Client() *s3.Client {
	r2_access_key_id := os.Getenv("R2_ACCESS_KEY_ID")
	r2_secret_key := os.Getenv("R2_SECRET_KEY")
	accountId := os.Getenv("CLOUDFLARE_ACCOUNT_ID")

	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountId),
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithRegion("us-east-1"),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(r2_access_key_id, r2_secret_key, "")),
	)
	if err != nil {
		panic(err)
	}

	client := s3.NewFromConfig(cfg)
	return client
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func gbToMb(gb int) int {
	return gb * 1024
}

func parseInt(s string) int {
	if s == "" {
		return 0
	}
	i, _ := strconv.Atoi(s)
	return i
}
