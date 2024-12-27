#!/bin/bash

TOKEN="${{ secrets.API_TOKEN }}"
REPO="${{ github.repository }}"
VALUE="${{ env.NEW_BUILD }}


curl -X PATCH \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Github-Api-Version: 2022-11-28" \
  "https://api.github.com/repos/$REPO/actions/variables/BUILD_NUMBER" \
  -d '{"value":"$VALUE"}'
