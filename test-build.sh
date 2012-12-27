#!/bin/bash
# Use the Go tool to build with a test version.
# Using this script is not recommended. Instead,
# put the following in your ~/.profile
#   alias go-buildt='go build -ldflags "-X main.minversion $(date -u +-%M%S)"'
go build -ldflags "-X main.minversion $(date -u +-%M%S)"