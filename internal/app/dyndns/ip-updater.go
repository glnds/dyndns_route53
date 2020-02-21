package dyndns

import (
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route54"

	"github.com/sirupsen/logrus"
)

// Response represents the API response from https://www.ipify.org/
type Response struct {
	IP string `json:"ip"`
}

// GetWanIP Call to ipify.org to obtain the host's WAM IP address.
func GetWanIP(log *logrus.Logger) string {
	url := "https://api.ipify.org?format=json"
	timeout := time.Duration(30 * time.Second)
	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Get(url)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var body Response
	if err := decoder.Decode(&body); err != nil {
		log.Fatalln(err)
	}
	log.Debugf("Current WAN ip: %s", body.IP)
	return body.IP
}

// GetFqdnIP Get the FQDN's current IP address
func GetFqdnIP(conf Config, log *logrus.Logger) string {
	ips, err := net.LookupHost(conf.Fqdn)
	if err != nil {
		log.Fatalln(err)
	}
	log.Debugf("Current ip bounded to '%s': %s", conf.Fqdn, ips[0])
	return ips[0]
}

// UpdateFqdnIP Update the FQDN with the current WAN IP address
func UpdateFqdnIP(conf Config, log *logrus.Logger, ip string) {
	var token string
	creds := credentials.NewStaticCredentials(conf.AccessKeyID, conf.SecretAccessKey, token)

	sess := session.Must(session.NewSession())
	svc := route53.New(sess, &aws.Config{Credentials: creds})

	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name:            aws.String(conf.Fqdn),
						Type:            aws.String("A"),
						ResourceRecords: []*route53.ResourceRecord{{Value: aws.String(ip)}},
						TTL:             aws.Int64(111),
					},
				},
			},
			Comment: aws.String("IP update by GO script"),
		},
		HostedZoneId: aws.String(conf.HostedZoneID),
	}
	resp, err := svc.ChangeResourceRecordSets(params)
	if err != nil {
		log.Fatalln(err)
	}

	// Pretty-print the response data.
	log.Debugf("Route53 response: %v", resp)
}
