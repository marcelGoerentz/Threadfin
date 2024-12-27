#!/bin/bash

curl -X PATCH \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer ${{ secrets.GITHUB_TOKEN }}" \
  -H "X-Github-Api-Version: 2022-11-28" \
  "https://api.github.com/repos/${{ github.repository }}/actions/variables/BUILD_NUMBER" \
  -d "{\"value\":\"${{ env.NEW_BUILD }}\"}"