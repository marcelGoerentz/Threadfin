#!/bin/bash

curl -X PATCH \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $1" \
  -H "X-Github-Api-Version: 2022-11-28" \
  "https://api.github.com/repos/$2/actions/variables/BUILD_NUMBER" \
  -d "{\"value\":\"$3\"}"
