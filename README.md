# Lambda Local Proxy

Lambda Local Proxy is a HTTP proxy that receives HTTP requests and invokes an AWS Lambda function. By combining with (lambci/docker-lambda)[https://github.com/lambci/docker-lambda], you can test Lambda handler code locally or on CI without deploying.

This proxy can emulate following event formats at this moment:

* Application Load Balancer (ALB)
* Application Load Balancer (ALB) with multi-value headers enabled


## Usage

```
Usage of lambda-local-proxy:
  -e string
        Lambda API endpoint
  -f string
        Lambda function name (default "myfunction")
  -l string
        HTTP listen address (default any)
  -m    Enable multi-value headers. Effective only with -t alb
  -p int
        HTTP listen port (default 8080)
  -t string
        HTTP gateway type ("alb" for ALB) (default "alb")

  Environment variables:
    AWS_REGION, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_SESSION_TOKEN
```

Once it starts, access `http://localhost:8080` to invoke a lambda function.

Examples:

```bash
# Invoking local code through lambci/docker-lambda
$ docker run --rm \
  -e DOCKER_LAMBDA_WATCH=1 -e DOCKER_LAMBDA_STAY_OPEN=1 -p 9001:9001 \
  -v "$PWD":/var/task:ro,delegated \
  lambci/lambda:python3.8 lambda_function.lambda_handler
$ lambda-local-proxy -e http://localhost:9001 -t alb
```

```bash
# Invoking a deployed Lambda function
$ export AWS_REGION=...
$ export AWS_ACCESS_KEY_ID=...
$ export AWS_SECRET_ACCESS_KEY=...
$ lambda-local-proxy -t alb
```

## Development

### Build

```
$ go build
```

### Release

See (GoReleaser)[https://goreleaser.com/] documents.

Cheat sheet:

```
# Build packages in ./dist
goreleaser --snapshot --skip-publish --rm-dist

# Release
git push         # make sure you pushed
git push --tags  # code and tags.
env GITHUB_TOKEN=... goreleaser
```

## License

Apache License, Version 2.0.
Copyright (C) 2020 Treasure Data Inc.

