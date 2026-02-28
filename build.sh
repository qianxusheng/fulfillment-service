#!/bin/bash
set -e

echo "Building Lambda function..."

GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o bootstrap cmd/lambda/main.go

echo "Packaging Lambda function..."
zip lambda.zip bootstrap

echo "Cleaning up..."
rm bootstrap

echo "✓ Lambda package created: lambda.zip"
