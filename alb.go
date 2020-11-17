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
)

func NewALBPayloadBuilder(enableMultiValue bool) PayloadBuilder {
    if enableMultiValue {
        return ALBMultiValuePayloadBuilder{}
    } else {
        return ALBPayloadBuilder{}
    }
}

type ALBPayloadBuilder struct {
}

type ALBMultiValuePayloadBuilder struct {
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

    event := struct {
        HttpMethod string `json:"httpMethod"`
        Path string `json:"path"`
        QueryStringParameters map[string]string `json:"queryStringParameters"`
        Headers map[string]string `json:"headers"`
        Body string `json:"body"`
        IsBase64Encoded bool `json:"isBase64Encoded"`
    }{
        HttpMethod: r.Method,
        Path: r.URL.Path,
        QueryStringParameters: ArrayMapToFirstElementMap(queryParams),
        Headers: ArrayMapToFirstElementMap(r.Header),
        Body: body,
        IsBase64Encoded: isBase64Encoded,
    }

    return json.Marshal(event)
}

func (pb ALBMultiValuePayloadBuilder) BuildRequest(r *http.Request) ([]byte, error) {
    queryParams, err := url.ParseQuery(r.URL.RawQuery)
    if err != nil {
        return nil, err
    }

    body, isBase64Encoded, err := ReadBodyAsString(r)
    if err != nil {
        return nil, err
    }

    event := struct {
        HttpMethod string `json:"httpMethod"`
        Path string `json:"path"`
        MultiValueQueryStringParameters map[string][]string `json:"multiValueQueryStringParameters"`
        MultiValueHeaders map[string][]string `json:"multiValueHeaders"`
        Body string `json:"body"`
        IsBase64Encoded bool `json:"isBase64Encoded"`
    }{
        HttpMethod: r.Method,
        Path: r.URL.Path,
        MultiValueQueryStringParameters: queryParams,
        MultiValueHeaders: r.Header,
        Body: body,
        IsBase64Encoded: isBase64Encoded,
    }

    return json.Marshal(event)
}

func (pb ALBPayloadBuilder) BuildResponse(blob []byte) (int, []byte, map[string][]string, error) {
    resp := struct {
        StatusCode int `json:"statusCode"`
        Headers map[string]string `json:"headers"`
        Body string `json:"body"`
        IsBase64Encoded bool `json:"isBase64Encoded"`
    }{}

    err := json.Unmarshal(blob, &resp)

    if err != nil {
        return BuildErrorResponse(err, "Invalid JSON response")
    } else if resp.IsBase64Encoded {
        binary, err := base64.StdEncoding.DecodeString(resp.Body)
        if err != nil {
            return BuildErrorResponse(err, "Invalid body in JSON response")
        }
        return resp.StatusCode, binary, MapToArrayMap(resp.Headers), nil
    } else {
        return resp.StatusCode, []byte(resp.Body), MapToArrayMap(resp.Headers), nil
    }
}

func (pb ALBMultiValuePayloadBuilder) BuildResponse(blob []byte) (int, []byte, map[string][]string, error) {
    resp := struct {
        StatusCode int `json:"statusCode"`
        MultiValueHeaders map[string][]string `json:"multiValueHeaders"`
        Body string `json:"body"`
        IsBase64Encoded bool `json:"isBase64Encoded"`
    }{}

    err := json.Unmarshal(blob, &resp)

    if err != nil {
        return BuildErrorResponse(err, "Invalid JSON response")
    } else if resp.IsBase64Encoded {
        binary, err := base64.StdEncoding.DecodeString(resp.Body)
        if err != nil {
            return BuildErrorResponse(err, "Invalid body in JSON response")
        }
        return resp.StatusCode, binary, resp.MultiValueHeaders, nil
    } else {
        return resp.StatusCode, []byte(resp.Body), resp.MultiValueHeaders, nil
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

