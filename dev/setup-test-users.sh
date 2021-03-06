#!/bin/bash
# This script...
#   Signs up {user1,user2}@example.com, both with a password of testing123
#   And adds example.com to user1@example.com's account
#   Makes user1@example.com an admin

echo -n Creating user1@example.com...
curl --request POST \
  --url http://localhost:8080/user/signup \
  --header 'Content-Type: application/json' \
  --data '{
        "email": "user1@example.com",
        "password": "testing123",
        "refer": "me"
}'
echo

echo -n "Enabling and making user1@example.com an admin..."
docker exec -it postgres psql --host localhost --username api --command "UPDATE users SET groups = '{core.ENABLED,core.ADMIN}' WHERE email = 'user1@example.com';"

echo -n Creating user2@example.com...
curl --request POST \
  --url http://localhost:8080/user/signup \
  --header 'Content-Type: application/json' \
  --data '{
        "email": "user2@example.com",
        "password": "testing123",
        "refer": "me"
}'
echo

echo -n Retreiving authentication token for user1@example.com...
AUTH_TOKEN=$(curl -s --request POST \
  --url http://localhost:8080/user/login \
  --header 'Content-Type: application/json' \
  --data '{
	"email": "user1@example.com",
	"password": "testing123"
}' | jq -r .data.token)
echo $AUTH_TOKEN

echo -n Creating example.com for user1@example.com...
curl --request POST \
  --url http://localhost:8080/dns/zones \
  --header "Authorization: Token $AUTH_TOKEN" \
  --header 'Content-Type: application/json' \
  --data '{"zone": "example.com"}'
echo
