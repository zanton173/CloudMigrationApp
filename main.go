package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joho/godotenv"
)

type initCloudFunction struct {
	optionOne config.LoadOptionsFunc
	optionTwo config.LoadOptionsFunc
}

type userCreds struct {
	cfaccountId  string
	r2apiKey     string
	r2apiSecret  string
	awsapikey    string
	awsapisecret string
}

type resolverConfig struct {
	acc      userCreds
	endpoint aws.EndpointResolverWithOptionsFunc
}

type listedObjectKeys []string

type bucket struct {
	name   string
	origin string
	prefix string
}

func (buc *bucket) reformatInput() {
	fmt.Scanln(&buc.name)
	buc.name = strings.ToLower(buc.name)
	countOfSep := strings.Count(buc.name, "/")
	if strings.ContainsAny(buc.name, "/") {
		if countOfSep == 1 && strings.Split(buc.name, "/")[1] == "" {
			buc.name = strings.Replace(buc.name, "/", "", 1)
			buc.origin = buc.name
		} else if countOfSep >= 1 && strings.Split(buc.name, "/")[1] != "" {
			findPrefix := strings.TrimRight(buc.name, "/")
			findOrigin := strings.SplitAfter(buc.name, "/")[0]
			buc.origin = strings.TrimRight(findOrigin, "/")
			buc.prefix = strings.Replace(findPrefix, findOrigin, "", 1)

		} else {
			fmt.Printf("bucketname %s\n", buc.name)
		}
	} else {
		buc.origin = buc.name
	}
}

var srcBucket bucket
var tgtBucket bucket

func main() {
	//time.Sleep(3 * time.Second)
	for {
		fmt.Println("What is your source bucket name?")
		srcBucket.reformatInput()
		fmt.Println("What is your target bucket name?")
		tgtBucket.reformatInput()

		var program_creds userCreds
		var r2resolve resolverConfig
		var awsresolve resolverConfig

		var srcCloudOptions initCloudFunction
		var tgtCloudOptions initCloudFunction

		program_creds.initCredentials()
		r2resolve.acc = program_creds
		r2resolve.endpoint = initBucketEndpoint("https://c4d32031aaebcfb281943747bd0cc8b4.r2.cloudflarestorage.com")

		awsresolve.acc = program_creds

		srcCloudOptions.optionOne = config.WithDefaultRegion("us-east-1")
		srcCloudOptions.optionTwo = config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(awsresolve.acc.awsapikey, awsresolve.acc.awsapisecret, ""))

		tgtCloudOptions.optionOne = config.WithEndpointResolverWithOptions(r2resolve.endpoint)
		tgtCloudOptions.optionTwo = config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(r2resolve.acc.r2apiKey, r2resolve.acc.r2apiSecret, ""))

		s3_aws_client := initCloudConfig(&awsresolve, srcCloudOptions)
		s3_r2_client := initCloudConfig(&r2resolve, tgtCloudOptions)

		//s3_r2_client, s3_aws_client := initCloudConfig(&awsresolve, &r2resolve)

		// ONLY USED TO SEND DATA TO SRC BUCKET
		// uploadFileToS3Temp(*s3_aws_client, formattedBucket)

		downloader := manager.NewDownloader(s3_aws_client)

		//listObjects(*s3_r2_client, r2BucketName)

		downloadToLocal(*s3_aws_client, srcBucket, *downloader)

		listFromStage := listObjectsFromStage()

		migrateFilesToR2S3(*s3_r2_client, tgtBucket, listFromStage)
	}
}

func (uc *userCreds) initCredentials() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
		os.Exit(1)
	}

	uc.cfaccountId = os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	uc.r2apiKey = os.Getenv("CLOUDFLARE_R2_API_KEY")
	uc.r2apiSecret = os.Getenv("CLOUDFLARE_R2_API_SECRET")
	uc.awsapikey = os.Getenv("AWS_API_KEY")
	uc.awsapisecret = os.Getenv("AWS_API_SECRET")

}

func initBucketEndpoint(cloudSpecificUrl string) aws.EndpointResolverWithOptionsFunc {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: cloudSpecificUrl,
		}, nil
	})
	return resolver
}
func initCloudConfig(resolvercfg *resolverConfig, ourOpts initCloudFunction) *s3.Client {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		ourOpts.optionOne,
		ourOpts.optionTwo,
	)

	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	}

	client := s3.NewFromConfig(cfg)
	return client
}

func listObjectsFromStage() []string {
	var output []string

	files, err := os.ReadDir(".")
	if err != nil {
		fmt.Println("Err finding directory downloads")
		fmt.Println(err)
		os.Exit(4)
	}
	for _, val := range files {
		output = append(output, val.Name())
	}

	return output
}
func listObjects(s3_client s3.Client, buc bucket) []string {
	var outputString listedObjectKeys
	var listObjectsOutput *s3.ListObjectsV2Output
	var err error
	if buc.prefix > "" {
		listObjectsOutput, err = s3_client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
			Bucket: &buc.origin,
			Prefix: &buc.prefix,
		})
		if err != nil {
			log.Fatal(err)
		}
	} else {
		listObjectsOutput, err = s3_client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
			Bucket: &buc.origin,
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, object := range listObjectsOutput.Contents {
		outputString = append(outputString, *object.Key)

	}
	return outputString
	// TODO - Move to listBuckets function
	/*
		listBucketsOutput, err := s3_client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})

		if err != nil {
			log.Fatal(err)
		}

		for _, object := range listBucketsOutput.Buckets {
			obj, _ := json.MarshalIndent(object, "", "\t")
			fmt.Println(string(obj))
		}
	*/
}

func downloadToLocal(s3_client s3.Client, buc bucket, dl manager.Downloader) {
	os.Mkdir("downloads", 0755)
	err := os.Chdir("downloads")
	if err != nil && !strings.ContainsAny(err.Error(), "downloads:") {
		os.Mkdir("downloads", 0755)
		os.Chdir("downloads")
	}
	os.Chdir("downloads")
	listOfObjKey := listObjects(s3_client, buc)
	formattedList := []string{}
	for i := range listOfObjKey {

		st, b := strings.CutSuffix(listOfObjKey[i], "/")
		if !b {
			formattedList = append(formattedList, st)
		}
	}
	for i := range formattedList {

		item := strings.SplitAfter(formattedList[i], "/")
		for _, val := range item {
			if !strings.ContainsAny(val, "/") {

				file, _ := os.Create(val)
				if err != nil && !strings.ContainsAny(err.Error(), "downloads:") {
					fmt.Printf("Error: %s", err)
				}
				defer file.Close()

				dl.Download(context.TODO(), file, &s3.GetObjectInput{
					Bucket: &buc.origin,
					Key:    &val,
				})

				fmt.Println("Downloaded ", file.Name())
			}
		}
	}
}

func migrateFilesToR2S3(r2_client s3.Client, buc bucket, listFromStage []string) {
	if buc.prefix > "" {
		for _, item := range listFromStage {
			_, err := r2_client.PutObject(context.TODO(), &s3.PutObjectInput{
				Bucket: &buc.origin,
				Key:    aws.String(buc.prefix + "/" + item),
			})
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("Moving %s to target bucket %s\n", item, buc.origin+"/"+buc.prefix)
		}
	} else {
		for _, item := range listFromStage {
			fmt.Println(item)
			_, err := r2_client.PutObject(context.TODO(), &s3.PutObjectInput{
				Bucket: &buc.origin,
				Key:    &item,
			})
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("Moving %s to target bucket %s\n", item, buc.origin)
		}
	}
	err := os.RemoveAll("downloads/")
	if err != nil {
		fmt.Printf("Error: %s removing files", err)
	}
}

/*
// Only used for mock file generation
func uploadFileToS3Temp(s3_client s3.Client, buc bucket) {

	for i := 0; i < 5; i++ {
		upFile, err := os.Open(fmt.Sprintf("./uploads/MyTestUpload%s", fmt.Sprint(i)) + ".txt")
		if err != nil {
			fmt.Println("Error on file:", err)
		}
		defer upFile.Close()

		upFileInfo, _ := upFile.Stat()
		var fileSize int64 = upFileInfo.Size()
		fileBuffer := make([]byte, fileSize)
		upFile.Read(fileBuffer)
		go s3Upload(s3_client, buc, fileBuffer, i, err)

	}

}

// Also part of mock file operations
func s3Upload(s3_client s3.Client, buc bucket, fileBuffer []byte, iter int, err error) {
	output, errput := s3_client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &buc.origin,
		// Body is for io.Reader
		Body: bytes.NewReader(fileBuffer),
		Key:  aws.String(fmt.Sprintf("MyTestUpload%s", fmt.Sprint(iter)) + ".txt"),
	})
	if errput != nil {
		fmt.Println("Erro on put: ", err)
	}
	fmt.Println(output)
}
*/
