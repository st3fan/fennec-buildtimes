#!/usr/bin/env bash

set -e
set -x

GOOS=linux go build main.go
zip main.zip main templates/main.html
#aws s3 mb s3://sarentz-buildtimes-artifacts
aws-sam-local package --template-file template.yaml --s3-bucket sarentz-buildtimes-artifacts --output-template-file package.yaml
aws-sam-local deploy --template-file ./package.yaml --stack-name sarentz-buildtimes-artifacts --capabilities CAPABILITY_IAM
