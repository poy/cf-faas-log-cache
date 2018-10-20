#!/bin/bash

set -eu

pwd=$PWD
PROJECT_DIR="$(cd "$(dirname "$0")/.."; pwd)"

app_name=""
cf_faas=""

function print_usage {
    echo "Usage: $0 [a:f:h]"
    echo " -a application name (REQUIRED) - The given name (and route) for CF-FaaS Log-Cache."
    echo " -f CF-FaaS address  (REQUIRED) - The address for CF-FaaS."
    echo " -h help                        - Shows this usage."
    echo
    echo "More information available at https://github.com/poy/cf-faas-log-cache"
}

function abs_path {
    case $1 in
        /*) echo $1 ;;
        *) echo $pwd/$1 ;;
    esac
}

function fail {
    echo $1
    exit 1
}

while getopts 'a:f:h' flag; do
  case "${flag}" in
    a) app_name="${OPTARG}" ;;
    f) cf_faas="${OPTARG}" ;;
    h) print_usage ; exit 1 ;;
  esac
done

# Ensure we are starting from the project directory
cd $PROJECT_DIR

if [ -z "$app_name" ]; then
    echo "AppName is required via -a flag"
    print_usage
    exit 1
fi

if [ -z "$cf_faas" ]; then
    echo "CF-FaaS is required via -f flag"
    print_usage
    exit 1
fi

# Clear any schema
cf_faas=$(echo ${cf_faas/https:\/\//})
cf_faas=$(echo ${cf_faas/http:\/\//})

TEMP_DIR=$(mktemp -d)

# CF-FaaS-Log-Cache binaries
echo "building CF-FaaS-Log-Cache binaries..."
GOOS=linux go build -o $TEMP_DIR/cf-faas-log-cache ./cmd/cf-faas-log-cache &> /dev/null || fail "failed to build cf-faas-log-cache"
cp cmd/cf-faas-log-cache/run.sh $TEMP_DIR
echo "done building CF-FaaS-Log-Cache binaries."

# CF-Space-Security binaries
echo "building CF-Space-Security binaries..."
go get github.com/poy/cf-space-security/... &> /dev/null || fail "failed to get cf-space-security"
GOOS=linux go build -o $TEMP_DIR/proxy ../cf-space-security/cmd/proxy &> /dev/null || fail "failed to build cf-space-security proxy"
GOOS=linux go build -o $TEMP_DIR/reverse-proxy ../cf-space-security/cmd/reverse-proxy &> /dev/null || fail "failed to build cf-space-security reverse proxy"
echo "done building CF-Space-Security binaries."

echo "pushing $app_name..."
cf push $app_name --no-start -p $TEMP_DIR -b binary_buildpack -c ./run.sh &> /dev/null || fail "failed to push app $app_name"
echo "done pushing $app_name."

if [ -z ${CF_HOME+x} ]; then
    CF_HOME=$HOME
fi

# Configure
echo "configuring $app_name..."
cf set-env $app_name REFRESH_TOKEN "$(cat $CF_HOME/.cf/config.json | jq -r .RefreshToken)" &> /dev/null || fail "failed to set REFRESH_TOKEN"
cf set-env $app_name CLIENT_ID "$(cat $CF_HOME/.cf/config.json | jq -r .UAAOAuthClient)" &> /dev/null || fail "failed to set set CLIENT_ID"
cf set-env $app_name CF_FAAS_ADDR "$cf_faas" &> /dev/null || fail "failed to set set CF_FAAS_ADDR"

skip_ssl_validation="$(cat $CF_HOME/.cf/config.json | jq -r .SSLDisabled)"
if [ $skip_ssl_validation = "true" ]; then
    cf set-env $app_name SKIP_SSL_VALIDATION true &> /dev/null || fail "failed to set SKIP_SSL_VALIDATION"
fi

echo "done configuring $app_name."

echo "starting $app_name..."
cf start $app_name &> /dev/null || fail "failed to start $app_name"
echo "done starting $app_name."
