#!/bin/bash
# This script signs up {user1,user2}@example.com, both a password of testing123, and adds example.com to user1@example.com's account

curl --request POST \
  --url http://localhost:8080/user/signup \
  --header 'Content-Type: application/json' \
  --data '{
        "email": "user1@example.com",
        "password": "testing123",
        "refer": "me"
}'

curl --request POST \
  --url http://localhost:8080/user/signup \
  --header 'Content-Type: application/json' \
  --data '{
        "email": "user2@example.com",
        "password": "testing123",
        "refer": "me"
}'

AUTH_TOKEN=$(curl -s --request POST \
  --url http://localhost:8080/user/login \
  --header 'Content-Type: application/json' \
  --data '{
	"email": "user1@example.com",
	"password": "testing123"
}' | jq -r .data.token)

curl --request POST \
  --url http://localhost:8080/dns/zones \
  --header "Authorization: Token $AUTH_TOKEN" \
  --header 'Content-Type: application/json' \
  --data '{"zone": "example.com"}'
