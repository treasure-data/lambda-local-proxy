package main

import (
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "strings"
    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/lambda"
)

type PayloadBuilder interface {
    BuildRequest(*http.Request) ([]byte, error)

    BuildResponse([]byte) (int, []byte, map[string][]string, error)
}

func main() {
    functionName := flag.String("f", "myfunction", "Lambda function name")
    endpoint := flag.String("e", "", "Lambda API endpoint")
    apiType := flag.String("t", "alb", "Gateway type (alb for ALB)")
    albMultiValue := flag.Bool("m", false, "Enable multi-value headers. Effective only with -t alb")
    flag.Usage = func() {
        fmt.Println("Usage of lambda-local-proxy:")
        flag.PrintDefaults()
        fmt.Println("")
        fmt.Println("  Environment variables:")
        fmt.Println("    AWS_REGION, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN")
    }
    flag.Parse()

    if *apiType != "alb" {
        fmt.Println("Unknown gateway type: " + *apiType)
        os.Exit(1)
    }

    pb := NewALBPayloadBuilder(*albMultiValue)
    client := MakeLambdaClient(*endpoint)
    handler := MakeInvokeLambdaHandler(client, *functionName, pb)

    http.HandleFunc("/", handler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func MakeLambdaClient(endpoint string) *lambda.Lambda {
    sess := session.Must(session.NewSessionWithOptions(session.Options{
        SharedConfigState: session.SharedConfigEnable,
    }))

    config := aws.Config{}
    if endpoint != "" {
        config.Endpoint = &endpoint
    }

    return lambda.New(sess, &config)
}

func MakeInvokeLambdaHandler(client *lambda.Lambda, functionName string, pb PayloadBuilder) func(http.ResponseWriter, *http.Request) {
    return func(w http.ResponseWriter, r *http.Request) {
        // Add proxy headers
        r.Header.Add("X-Forwarded-For", r.RemoteAddr[0:strings.LastIndex(r.RemoteAddr, ":")])
        r.Header.Add("X-Forwarded-Proto", "http")
        r.Header.Add("X-Forwarded-Port", "8080")

        // Parse HTTP response and create an event
        payload, err := pb.BuildRequest(r)
        if err != nil {
            WriteErrorResponse(w, "Invalid request", err)
            return
        }

        // Invoke Lambda with the event
        output, err := client.Invoke(&lambda.InvokeInput{
            FunctionName: aws.String(functionName),
            Payload: payload,
        })
        if err != nil {
            WriteErrorResponse(w, "Failed to invoke Lambda", err)
            return
        }
        if output.FunctionError != nil {
            WriteErrorResponse(w, "Lambda function error: " + *output.FunctionError, nil)
            return
        }

        // Build a response
        status, body, headers, err := pb.BuildResponse(output.Payload)
        if err != nil {
            WriteErrorResponse(w, "Invalid JSON response", err)
            return
        }

        // Write the response - headers, status code, and body
        for key, values := range headers {
            for _, value := range values {
                w.Header().Add(key, value)
            }
        }
        w.WriteHeader(status)
        w.Write(body)
        return
    }
}

func WriteErrorResponse(w http.ResponseWriter, message string, err error) {
    body := "502 Bad Gateway\n" + message
    if err != nil {
        body += "\n" + err.Error()
    }
    w.WriteHeader(502)  // Bad Gateway
    w.Write([]byte(body))
    return
}
