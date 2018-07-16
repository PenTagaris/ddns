package main

import (
	"fmt"
    "encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

type RequestBody struct {
    NewIP string `json:"ip_address"`
    HostedZone string `json:"hosted_zone"`
    TargetURL string `json:"target_url"`
}

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

func ParseBody(body []byte) (string, string, string, error) {
    //Data is going to be our json struct
    data := RequestBody{}

    //Unmarshal the string to our json struct, and fail out if need be
    err := json.Unmarshal(body, data)
    if err != nil {
        return "", "", "", err
    }

    //else, return our data
    return data.NewIP, data.HostedZone, data.TargetURL, err
}

//Lots of errors to deal with, maybe need a custom handler?
func ErrorHandler (statusCode int, errorText string, err error) (events.APIGatewayProxyResponse) {
    //TODO: give more info, maybe better headers?
    return events.APIGatewayProxyResponse{
        StatusCode: statusCode,
        Body:       errorText + err.Error(),
        Headers: map[string]string{
            "Content-Type": "text/html",
        },
    }
}
// Handler is executed by AWS Lambda in the main function. Once the request
// is processed, it returns an Amazon API Gateway response object to AWS Lambda
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

    //Caller is the X-Forwarded-For header from Cloudfront 
    //request.Body should be json, so byte encode it here and let the parser do its thing
	caller := string(request.RequestContext.Identity.SourceIP)
    newIP, hostedZone, targetURL, parseErr := ParseBody([]byte(request.Body))

    //Break if we get an error while parsing
    if parseErr != nil {
        return ErrorHandler(500, "Parsing Error", err), err
    }
    //Also break if the X-F-F header doesn't match newIP
    else if caller != newIP {
	    return ErrorHandler(500, "Failed to Validate", err), err
    }

    //Here's where we actually make the update to R53
    result, err := updateR53(newIP, hostedZone, targetURL)

    //Print out our body for testing purposes
    fmt.Printf("Body from the request: %+v", body)

    //Log the result
    fmt.Printf("Result of the call %+v", result)

    //If we get an error from the update itself, 
    //just do a general 500 and send the problem to the caller
    if err != nil {
        return ErrorHandler(500, "Update Failed", err), err
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
