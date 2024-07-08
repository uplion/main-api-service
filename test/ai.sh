#!/usr/bin/env bash
stream=false
if [ "$1" == "stream" ]; then
  stream=true
fi

data=$(jo -p model=gpt-3.5-turbo" messages=$(jo -a $(jo role="user" content="Hello!")) stream=$stream)

curl -X POST -H "Content-Type: application/json" -d "$data" http://localhost:8080/api/v1/chat/completions -v