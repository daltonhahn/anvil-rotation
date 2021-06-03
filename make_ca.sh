#!/bin/bash
mkdir config
openssl req -x509 -sha256 -nodes -days 365 -newkey rsa:4096 -keyout config/ca.key -out config/ca.crt
