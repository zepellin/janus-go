package main

import (
	"log"
	"net/http"
	"time"
	"fmt"
	"os"
	"io/ioutil"

	"cloud.google.com/go/compute/metadata"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/awserr"
)

func truncateText(s string, max int) string {
    return s[:max]
}

func gcpMetadataClient() (*metadata.Client) {
        c := metadata.NewClient(&http.Client{Timeout: 1 * time.Second, Transport: userAgentTransport{
                userAgent: "my-user-agent",
                base:      http.DefaultTransport,
        }})

        return c
}

// This example demonstrates how to use your own transport when using this package.
func main() {
        c := gcpMetadataClient()


	projectId, err := c.ProjectID()
	if err != nil {
		log.Fatal("Couldn't connect to metadata server")
	}

        hostname, err := c.Hostname()
        if err != nil {
                log.Fatal("Couldn't connect to metadata server")
        }

	client := &http.Client{}
	req, _ := http.NewRequest("GET", "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity?format=standard&audience=gcp", nil)
	req.Header.Set("Metadata-Flavor", "Google")
	response, _ := client.Do(req)

	body, error := ioutil.ReadAll(response.Body)
	if error != nil {
	   fmt.Println(error)
	}
	response.Body.Close()

        if err != nil {
        	log.Fatal("Couldn't connect to metadata server")
        }


        sessionIdentifier := fmt.Sprintf("%s-%s", projectId, hostname)[:64]

        //fmt.Println(sessionIdentifier)
	//fmt.Println(string(body))
        fmt.Println(os.Args[1])


	// Here we start with AWS stuff
        svc := sts.New(session.New())

	input := &sts.AssumeRoleWithWebIdentityInput{
            RoleArn:         aws.String(os.Args[1]),
            RoleSessionName:  aws.String(sessionIdentifier),
            WebIdentityToken: aws.String(string(body)),
            DurationSeconds:  aws.Int64(3600),
        }
        result, err := svc.AssumeRoleWithWebIdentity(input)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case sts.ErrCodeMalformedPolicyDocumentException:
				fmt.Println(sts.ErrCodeMalformedPolicyDocumentException, aerr.Error())
			case sts.ErrCodePackedPolicyTooLargeException:
				fmt.Println(sts.ErrCodePackedPolicyTooLargeException, aerr.Error())
			case sts.ErrCodeIDPRejectedClaimException:
				fmt.Println(sts.ErrCodeIDPRejectedClaimException, aerr.Error())
			case sts.ErrCodeIDPCommunicationErrorException:
				fmt.Println(sts.ErrCodeIDPCommunicationErrorException, aerr.Error())
			case sts.ErrCodeInvalidIdentityTokenException:
				fmt.Println(sts.ErrCodeInvalidIdentityTokenException, aerr.Error())
			case sts.ErrCodeExpiredTokenException:
				fmt.Println(sts.ErrCodeExpiredTokenException, aerr.Error())
			case sts.ErrCodeRegionDisabledException:
				fmt.Println(sts.ErrCodeRegionDisabledException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}

	fmt.Println(result)
}

// userAgentTransport sets the User-Agent header before calling base.
type userAgentTransport struct {
	userAgent string
	base      http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface.
func (t userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.userAgent)
	return t.base.RoundTrip(req)
}

