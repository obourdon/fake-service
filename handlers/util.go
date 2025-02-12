package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nicholasjackson/fake-service/client"
	"github.com/nicholasjackson/fake-service/grpc/api"
	"github.com/nicholasjackson/fake-service/logging"
	"github.com/nicholasjackson/fake-service/response"
	"github.com/nicholasjackson/fake-service/worker"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/patrickmn/go-cache"
	"google.golang.org/grpc/status"
)

const timeFormat = "2006-01-02T15:04:05.000000"

var cloud_meta_cache *cache.Cache

type AWSCloudInfos struct {
	Provider         string `json:"provider,omitempty"`
	InstanceID       string `json:"instanceId,omitempty"`
	InstanceType     string `json:"instanceType,omitempty"`
	PrivateIP        string `json:"privateIp,omitempty"`
	AvailabilityZone string `json:"availabilityZone,omitempty"`
}

func workerHTTP(ctx opentracing.SpanContext, uri string, defaultClient client.HTTP, pr *http.Request, l *logging.Logger, content []byte) (*response.Response, error) {
	httpReq, _ := http.NewRequest(http.MethodGet, uri, nil)
	if len(content) > 0 {
		httpReq, _ = http.NewRequest(http.MethodPost, uri, bytes.NewReader(content))
	}

	hr := l.CallHTTPUpstream(pr, httpReq, ctx)
	defer hr.Finished()

	code, resp, headers, cookies, err := defaultClient.Do(httpReq, pr)

	hr.SetMetadata("response", strconv.Itoa(code))
	hr.SetError(err)

	r := &response.Response{}

	if resp != nil {
		jsonerr := r.FromJSON(resp)
		if jsonerr != nil {
			// we can not process the upstream response
			// this could be because the proxy is returning an error not the
			// upstream
			// in this instance create a blank response with the error
			l.Log().Error("Unable to read response JSON", "error", jsonerr)
		}
	}

	// set the local URI for the upstream
	r.URI = uri
	r.Code = code
	r.Headers = headers
	r.Cookies = cookies

	if err != nil {
		r.Error = err.Error()
	}

	return r, err
}

func workerGRPC(ctx opentracing.SpanContext, uri string, grpcClients map[string]client.GRPC, l *logging.Logger, content []byte) (*response.Response, error) {
	hr, outCtx := l.CallGRCPUpstream(uri, ctx)
	defer hr.Finished()

	c := grpcClients[uri]
	resp, headers, err := c.Handle(outCtx, &api.Request{Data: content})

	r := &response.Response{}
	if err != nil {
		r.Error = err.Error()
		hr.SetError(err) // set the error for logging

		if s, ok := status.FromError(err); ok {
			r.Code = int(s.Code())
			hr.SetMetadata("ResponseCode", strconv.Itoa(r.Code)) // set the response code for logging

			// response will always be nil when an error has occured, check to see if we can get the details from the
			// error message
			if len(s.Details()) > 0 {
				if d, ok := s.Details()[0].(*api.Response); ok {
					r.FromJSON([]byte(d.Message))
				}
			}
		}
	}

	if resp != nil {
		jsonerr := r.FromJSON([]byte(resp.Message))
		if jsonerr != nil {
			// we can not process the upstream response
			// this could be because the proxy is returning an error not the
			// upstream
			// in this instance create a blank response with the error
			l.Log().Error("Unable to read response JSON", "error", jsonerr)
		}
	}

	// set the local URI for the upstream
	r.URI = uri
	r.Type = "gRPC"
	r.Headers = headers

	if err != nil {
		r.Error = err.Error()
		return r, err
	}

	return r, nil
}

func processResponses(responses []worker.Done) []byte {
	respLines := []string{}

	// append the output from the upstreams
	for _, r := range responses {
		respLines = append(respLines, fmt.Sprintf("## Called upstream uri: %s", r.URI))
		/*
			// indent the reposne from the upstream
			lines := strings.Split(r.Message, "\n")
			for _, l := range lines {
				respLines = append(respLines, fmt.Sprintf("  %s", l))
			}
		*/
	}

	return []byte(strings.Join(respLines, "\n"))
}

// Get a list of non loopback Ip addresses
// realistically this is not going to change to cache
var ipAddresses []string

func getIPInfo() []string {
	if len(ipAddresses) > 0 {
		return ipAddresses
	}

	ips := []string{}

	ifaces, _ := net.Interfaces()
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {

			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// ignore localhost
			if !ip.IsLoopback() && !ip.IsMulticast() && ip.To4() != nil {
				ips = append(ips, ip.String())
			}
			// process IP address
		}
	}

	// cache the result
	ipAddresses = ips
	return ips
}

func getHostname() string {
	h, _ := os.Hostname()
	return h
}

func getURL(url string, headers [][]string) ([]byte, error) {
	cloudMetdataAPIClient := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for i := range headers {
		fmt.Println(i)
		req.Header.Set(headers[i][0], headers[i][1])
	}
	res, err := cloudMetdataAPIClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.Body != nil {
		defer res.Body.Close()
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return body, err
	}
	return body, nil
}

func getAzureMetadata() (response.CloudInfos, error) {
	cInfos := response.CloudInfos{}
	_, err := getURL(
		"http://169.254.169.254/metadata/instance?api-version=2020-06-01",
		[][]string{
			{"User-Agent", "fake-service"},
			{"Metadata", "true"},
		},
	)
	if err != nil {
		return cInfos, err
	}
	return cInfos, nil
}

func getAWSMetadata() (response.CloudInfos, error) {
	cInfos := response.CloudInfos{}
	awsInfos := AWSCloudInfos{}
	body, err := getURL(
		"http://169.254.169.254/latest/dynamic/instance-identity/document",
		[][]string{
			{"User-Agent", "fake-service"},
		},
	)
	if err != nil {
		return cInfos, err
	}
	err = json.Unmarshal(body, &awsInfos)
	if err != nil {
		return cInfos, err
	}
	awsInfos.Provider = "aws"
	return response.CloudInfos{
		awsInfos.Provider,
		awsInfos.InstanceID,
		awsInfos.InstanceType,
		awsInfos.PrivateIP,
		awsInfos.AvailabilityZone,
	}, nil
}

func retrieveCloudInfos() response.CloudInfos {
	ci, err := getAWSMetadata()
	if err != nil {
		ci, err = getAzureMetadata()
		if err != nil {
			return response.CloudInfos{}
		}
	}
	return ci
}

func getCloudInfos() response.CloudInfos {
	if cloud_meta_cache == nil {
		return response.CloudInfos{}
	}
	foo, found := cloud_meta_cache.Get("cloudMetaInfos")
	if found {
		return foo.(response.CloudInfos)
	}
	newInfos := retrieveCloudInfos()
	cloud_meta_cache.Set("cloudMetaInfos", newInfos, cache.DefaultExpiration)
	return newInfos
}

func InitCloudMetadataCache(gatherCloudMetadata bool) {
	cloud_meta_cache = cache.New(1*time.Minute, 2*time.Minute)
	if gatherCloudMetadata {
		cloud_meta_cache.Set("cloudMetaInfos", retrieveCloudInfos(), cache.DefaultExpiration)
	} else {
		cloud_meta_cache.Set("cloudMetaInfos", response.CloudInfos{}, cache.NoExpiration)
	}
}
