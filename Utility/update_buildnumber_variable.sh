#!/bin/bash

GITHUB_TOKEN=$1
REPOSITORY=$2
NEW_BUILD=$3

curl -X PATCH \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $GITHUB_TOKEN" \
  -H "X-Github-Api-Version: 2022-11-28" \
  "https://api.github.com/repos/$REPOSITORY/actions/variables/BUILD_NUMBER" \
  -d '{"value":"$NEW_BUILD"}'
