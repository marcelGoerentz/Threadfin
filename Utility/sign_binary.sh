#!/bin/bash

openssl dgst -sha256 -sign threadfin_privkey.pem -out signature.bin threadfin
cat signature.bin >> threadfin
