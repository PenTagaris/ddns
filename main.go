package main

import (
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

func updateR53(newIP string, hostedZone string, targetURL string) (*route53.ChangeResourceRecordSetsOutput, error) {
	// New service handler
	svc := route53.New(session.New())

	//Our input. It's really nested.
	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(targetURL),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(newIP),
							},
						},
						TTL:  aws.Int64(60),
						Type: aws.String("A"),
					},
				},
			},
			Comment: aws.String(fmt.Sprintf("Update to %s in hosted zone %s called from %s", targetURL, hostedZone, newIP)),
		},
		HostedZoneId: aws.String(hostedZone),
	}

	result, err := svc.ChangeResourceRecordSets(input)
	if err != nil {
		fmt.Println(err.Error())
	}
	return result, err
}

// Handler is executed by AWS Lambda in the main function. Once the request
// is processed, it returns an Amazon API Gateway response object to AWS Lambda
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

    //This is basically the X-Forwarded-For header from Cloudfront, and is our best
    //indicator for who called this
    //TODO: Make sure the body and the SourceIP match
	caller := string(request.RequestContext.Identity.SourceIP)
    hostedZone := string("Z1N0R6CQ9D3SXO")
    targetURL := string("home.christiannet.info")

    //Here's where we actually make the update to R53
    result, err := updateR53(caller, hostedZone, targetURL)

    //Always print the result
    fmt.Println(result)

    //If we get an error, just do a general 500 and send the problem to the caller
    if err != nil {
	    return events.APIGatewayProxyResponse{
		    StatusCode: 500,
            Body:       string("Error: " + err.Error()),
		    Headers: map[string]string{
			    "Content-Type": "text/html",
		    },
	    }, err
    }


	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string("This API endpoint was called from " + caller),
		Headers: map[string]string{
			"Content-Type": "text/html",
		},
	}, nil

}

func main() {
	lambda.Start(Handler)
}
