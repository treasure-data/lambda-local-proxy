package main

import (
    "bytes"
    "encoding/base64"
    "encoding/json"
    "io"
    "mime"
    "net/http"
    "net/url"
    "strings"
    "github.com/aws/aws-lambda-go/events"
)

func NewALBPayloadBuilder(enableMultiValue bool) PayloadBuilder {
    return ALBPayloadBuilder{
        EnableMultiValue: enableMultiValue,
    }
}

type ALBPayloadBuilder struct {
    EnableMultiValue bool
}

func (pb ALBPayloadBuilder) BuildRequest(r *http.Request) ([]byte, error) {
    queryParams, err := url.ParseQuery(r.URL.RawQuery)
    if err != nil {
        return nil, err
    }

    body, isBase64Encoded, err := ReadBodyAsString(r)
    if err != nil {
        return nil, err
    }

    event := events.ALBTargetGroupRequest{
        HTTPMethod: r.Method,
        Path: r.URL.Path,
        Body: body,
        IsBase64Encoded: isBase64Encoded,
        RequestContext: events.ALBTargetGroupRequestContext{
            ELB: events.ELBContext{
                TargetGroupArn: "arn:aws:elasticloadbalancing:us-east-2:123456789012:targetgroup/lambda-local-proxy-dummy/1234567890123456",
            },
        },
    }

    if pb.EnableMultiValue {
        event.MultiValueQueryStringParameters = queryParams
        event.MultiValueHeaders = r.Header
    } else {
        event.QueryStringParameters = ArrayMapToFirstElementMap(queryParams)
        event.Headers = ArrayMapToFirstElementMap(r.Header)
    }

    return json.Marshal(event)
}

func (pb ALBPayloadBuilder) BuildResponse(blob []byte) (int, []byte, map[string][]string, error) {
    resp := events.ALBTargetGroupResponse{}

    err := json.Unmarshal(blob, &resp)

    var body []byte
    if err != nil {
        return BuildErrorResponse(err, "Invalid JSON response")
    } else if resp.IsBase64Encoded {
        binary, err := base64.StdEncoding.DecodeString(resp.Body)
        if err != nil {
            return BuildErrorResponse(err, "Invalid body in JSON response")
        }
        body = binary
    } else {
        body = []byte(resp.Body)
    }

    if pb.EnableMultiValue {
        return resp.StatusCode, body, resp.MultiValueHeaders, nil
    } else {
        return resp.StatusCode, body, MapToArrayMap(resp.Headers), nil
    }
}

func ReadBodyAsString(r *http.Request) (string, bool, error) {
    if IsTextContentType(r) {
        body, err := ReadFullyAsString(r.Body)
        return body, false, err
    } else {
        binary, err := ReadFullyAsBinary(r.Body)
        if err != nil {
            return "", false, err
        }
        body := base64.StdEncoding.EncodeToString(binary)
        return body, true, nil
    }
}

func ArrayMapToFirstElementMap(arrayMap map[string][]string) map[string]string {
    m := make(map[string]string)
    for key, values := range arrayMap {
        m[key] = values[0]
    }
    return m
}

func MapToArrayMap(m map[string]string) map[string][]string {
    arrayMap := make(map[string][]string)
    for key, value := range m {
        arrayMap[key] = []string { value }
    }
    return arrayMap
}

func ReadFullyAsBinary(reader io.Reader) ([]byte, error) {
    buf := new(bytes.Buffer)
    _, err := buf.ReadFrom(reader)
    return buf.Bytes(), err
}

func ReadFullyAsString(reader io.Reader) (string, error) {
    buf := new(bytes.Buffer)
    _, err := buf.ReadFrom(reader)
    return buf.String(), err
}

func IsTextContentType(r *http.Request) bool {
    contentType := r.Header.Get("Content-type")

    t, _, err := mime.ParseMediaType(contentType)
    if err != nil {
        return false
    }

    return strings.HasPrefix(t, "text/") || t == "application/json" || t == "application/javascript" || t == "application/xml"
}

func BuildErrorResponse(err error, message string) (int, []byte, map[string][]string, error) {
    body := "502 Bad Gateway\n" + message + "\n" + err.Error()
    return 502, []byte(body), make(map[string][]string), err
}

