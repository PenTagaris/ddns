This is a DDNS for AWS services. This Lambda function is written in golang, calls Route 53, and receives data rom API Gateway. It takes a json POST payload in the following format:
```
{
    "ip_address": "TARGET_IP_ADDRESS",
    "hosted_zone":"HOSTED_ZONE",
    "target_URL":"URL_TO_UPDATE" 
}
```
and makes a call to Route 53 to update the $URL_TO_UPDATE with the $TARGET_IP_ADDRESS. The $HOSTED_ZONE needs to exist, but because it uses UPSERT, the $URL_TO_UPDATE does not need to exist at first.

Coding quality in here is probably rather low, since I'm using this to teach myself golang.
