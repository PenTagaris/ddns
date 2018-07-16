package main

import (
	"fmt"
    "encoding/json"
    "errors"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

type requestBody struct {
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

func parseBody(body []byte) (string, string, string, error) {
    //Data is going to be our json struct
    data := requestBody{}

    //Unmarshal the string to our json struct, and fail out if need be
    err := json.Unmarshal(body, &data)
    if err != nil {
        return "", "", "", err
    }

    //else, return our data
    return data.NewIP, data.HostedZone, data.TargetURL, err
}

//TODO: Implement Error Types
func errorHandler (statusCode int, errorString string) (events.APIGatewayProxyResponse, error) {
    //TODO: give more info, maybe better headers?
    return events.APIGatewayProxyResponse{
        StatusCode: int(statusCode),
        Body:       string(errorString),
        Headers:    map[string]string{
            "Content-Type": "text/html",
        },
    }, errors.New("My very own " + errorString)
}

//Handler is where the Lambda magic happens
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

    //We need a Body and a Source IP. If we don't have both, fail out
    if (request.RequestContext.Identity.SourceIP == "") || (request.Body == "") {
        //return errorHandler(500, "Not enough data to update")
        return events.APIGatewayProxyResponse{
            StatusCode: 400,
            Body:       string("Not enough data to update"),
            Headers:    map[string]string{
                "Content-Type": "text/html",
            },
        }, nil
    }
    //Print out our body for logging purposes
    fmt.Printf("Body from the request: %+v\n", request.Body)

    //Caller is the X-Forwarded-For header from Cloudfront 
    //request.Body should be json, so byte encode it here and let the parser do its thing
	caller := string(request.RequestContext.Identity.SourceIP)
    newIP, hostedZone, targetURL, parseErr := parseBody([]byte(request.Body))

    //error out if we don't have all three defined
    if (newIP == "") ||  (hostedZone == "") || (targetURL == "") {
        return errorHandler(400, "Not enough data to update")
    }
    //Break if we get an error while parsing
    if parseErr != nil {
       return errorHandler(500, parseErr.Error())

    //Also break if the X-F-F header doesn't match newIP
    } else if caller != newIP {
	    return errorHandler(500, "Unable to validate")
    }

    //Here's where we actually make the update to R53
    result, err := updateR53(newIP, hostedZone, targetURL)

    //Log the result
    fmt.Printf("Result of the call %+v\n", result)

    //If we get an error from the update itself, 
    //just do a general 500 and send the problem to the caller
    if err != nil {
        return errorHandler(500, err.Error())
    }

	return events.APIGatewayProxyResponse{
		StatusCode: 202,
		Body:       string("Call to update accepted"),
		Headers: map[string]string{
			"Content-Type": "text/html",
		},
	}, nil

}

func main() {
	lambda.Start(Handler)
}
