@echo off
echo Building Lambda function...

set GOOS=linux
set GOARCH=amd64
go build -tags lambda.norpc -o bootstrap cmd/lambda/main.go

echo Packaging Lambda function...
tar -a -c -f lambda.zip bootstrap

echo Cleaning up...
del bootstrap

echo Done! Lambda package created: lambda.zip
