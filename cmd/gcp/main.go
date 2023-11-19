package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"golang.org/x/oauth2"
	"google.golang.org/api/iamcredentials/v1"
	"google.golang.org/api/option"
	"google.golang.org/api/sts/v1"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func getGcpCredentials(worloadIdentityPool string, gcpserviceaccount string, stsRegion string) {

	ctx := context.Background()

	awsRoleCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(stsRegion))
	if err != nil {
		logger.Error("failed to load default AWS config, %s" + err.Error())
		os.Exit(1)
	}

	awsRoleCredentials, err := awsRoleCfg.Credentials.Retrieve(ctx)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	req, _ := http.NewRequest("POST", "https://sts."+stsRegion+".amazonaws.com?Action=GetCallerIdentity&Version=2011-06-15", nil)
	req.Header.Add("X-Goog-Cloud-Target-Resource", worloadIdentityPool)

	requestSignerWithToken := &awsRequestSigner{
		RegionName: stsRegion,
		AwsSecurityCredentials: awsSecurityCredentials{
			AccessKeyID:     awsRoleCredentials.AccessKeyID,
			SecretAccessKey: awsRoleCredentials.SecretAccessKey,
			SecurityToken:   awsRoleCredentials.SessionToken,
		},
	}

	err = requestSignerWithToken.SignRequest(req)
	if err != nil {
		logger.Error(err.Error())
	}

	awsSignedReq := awsRequest{
		URL:    req.URL.String(),
		Method: "POST",
	}
	for headerKey, headerList := range req.Header {
		for _, headerValue := range headerList {
			awsSignedReq.Headers = append(awsSignedReq.Headers, awsRequestHeader{
				Key:   headerKey,
				Value: headerValue,
			})
		}
	}
	sort.Slice(awsSignedReq.Headers, func(i, j int) bool {
		headerCompare := strings.Compare(awsSignedReq.Headers[i].Key, awsSignedReq.Headers[j].Key)
		if headerCompare == 0 {
			return strings.Compare(awsSignedReq.Headers[i].Value, awsSignedReq.Headers[j].Value) < 0
		}
		return headerCompare < 0
	})

	awsSubjectToken, err := json.Marshal(awsSignedReq)

	stsExchangeTokenRequest := sts.GoogleIdentityStsV1ExchangeTokenRequest{
		GrantType:          "urn:ietf:params:oauth:grant-type:token-exchange",
		RequestedTokenType: "urn:ietf:params:oauth:token-type:access_token",
		SubjectTokenType:   "urn:ietf:params:aws:token-type:aws4_request",
		Audience:           worloadIdentityPool,
		Scope:              "https://www.googleapis.com/auth/cloud-platform",
		SubjectToken:       url.QueryEscape(string(awsSubjectToken)),
	}

	gcpStsService, err := sts.NewService(ctx, option.WithoutAuthentication())
	if err != nil {
		logger.Error("ERR gcpStsService")
	}

	gcpStsV1Service := sts.NewV1Service(gcpStsService)

	stsToken, err := gcpStsV1Service.Token(&stsExchangeTokenRequest).Do()
	if err != nil {
		logger.Error(err.Error())
	}

	stsOauthToken := oauth2.Token{
		AccessToken: stsToken.AccessToken,
		TokenType:   stsToken.TokenType,
	}

	config := &oauth2.Config{}
	iamCredentialsService, err := iamcredentials.NewService(ctx, option.WithTokenSource(config.TokenSource(ctx, &stsOauthToken)))
	if err != nil {
		logger.Error(err.Error())
	}

	accessTokenRequest := iamcredentials.GenerateAccessTokenRequest{
		// Lifetime: "3600 second",
		Scope: []string{"https://www.googleapis.com/auth/cloud-platform"},
	}

	gcpCredentials, err := iamCredentialsService.Projects.ServiceAccounts.GenerateAccessToken("projects/-/serviceAccounts/"+gcpserviceaccount, &accessTokenRequest).Do()
	if err != nil {
		logger.Error(err.Error())
	}

	fmt.Println(gcpCredentials.AccessToken)
	fmt.Println(gcpCredentials.ExpireTime)
	some, _ := gcpCredentials.MarshalJSON()
	fmt.Println(string(some))
}
