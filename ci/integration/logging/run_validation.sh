#!/bin/env bash
# CI script for CentOS8 job
# purpose: verify the expected logging data is saved in supported storage types


set -ex

yum install -y jq hostname

TS=$(date +'%Y.%m.%d')
HOST=$(hostname)

ELASTIC_URL=http://127.0.0.1:9200
LOKI_URL=http://127.0.0.1:3100

yum install -y jq

######################### validate elasticsearch data #########################
# debug output of cluster status
curl -sX GET "$ELASTIC_URL/_cluster/health?pretty"
# verify expected index
#TODO(mmagr): adapt elasticseatch plugin to create index templates avoinding unnecessary prefix and suffix
expected_index="sglogs-$(echo $HOST | tr - _).$TS"
curl -sX GET "$ELASTIC_URL/_cat/indices/sglogs-*?h=index"
found_index=$(curl -sX GET "$ELASTIC_URL/_cat/indices/sglogs-*?h=index")
[[ "${found_index}" =~ "${expected_index}" ]] || exit 1
# debug output of index content
curl -sX GET "$ELASTIC_URL/${expected_index}/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "query": {
    "match_all": {}
  }
}
'
# verify expected documents
res=$(curl -sX GET "$ELASTIC_URL/${expected_index}/_search" -H 'Content-Type: application/json' -d"
{
  \"query\": {
    \"match_phrase\": {
      \"message\": {
        \"query\": \"WARNING Something bad might happen\"
      }
    }
  }
}
")
[ "$(echo $res | jq -r .hits.hits[0]._source.labels.severity)" = "warning" ] || exit 1
[ "$(echo $res | jq -r .hits.hits[0]._source.labels.file)" = "/tmp/test.log" ] || exit 1
[ "$(echo $res | jq -r .hits.hits[0]._source.labels.host)" = "$HOST" ] || exit 1
[ "$(echo $res | jq -r .hits.hits[0]._source.labels.tag)" = "ci.integration.test" ] || exit 1

res=$(curl -sX GET "$ELASTIC_URL/${expected_index}/_search" -H 'Content-Type: application/json' -d"
{
  \"query\": {
    \"match_phrase\": {
      \"message\": {
        \"query\": \":ERROR: Something bad happened\"
      }
    }
  }
}
")
[ "$(echo $res | jq -r .hits.hits[0]._source.labels.severity)" = "critical" ] || exit 1
[ "$(echo $res | jq -r .hits.hits[0]._source.labels.file)" = "/tmp/test.log" ] || exit 1
[ "$(echo $res | jq -r .hits.hits[0]._source.labels.host)" = "$HOST" ] || exit 1
[ "$(echo $res | jq -r .hits.hits[0]._source.labels.tag)" = "ci.integration.test" ] || exit 1

res=$(curl -sX GET "$ELASTIC_URL/${expected_index}/_search" -H 'Content-Type: application/json' -d"
{
  \"query\": {
    \"match_phrase\": {
      \"message\": {
        \"query\": \"[DEBUG] Wubba lubba dub dub\"
      }
    }
  }
}
")
[ "$(echo $res | jq -r .hits.hits[0]._source.labels.severity)" = "debug" ] || exit 1
[ "$(echo $res | jq -r .hits.hits[0]._source.labels.file)" = "/tmp/test.log" ] || exit 1
[ "$(echo $res | jq -r .hits.hits[0]._source.labels.host)" = "$HOST" ] || exit 1
[ "$(echo $res | jq -r .hits.hits[0]._source.labels.tag)" = "ci.integration.test" ] || exit 1

############################## validate loki data #############################
# verify expected labels
found_labels=$(curl -sX GET "$LOKI_URL/loki/api/v1/labels" | jq .data)
expected_labels='[
  "__name__",
  "cloud",
  "facility",
  "file",
  "host",
  "region",
  "severity",
  "source",
  "tag"
]'
[ "${expected_labels}" = "${found_labels}" ] || exit 1
# verify expected messages
curl -sG "$LOKI_URL/loki/api/v1/query" --data-urlencode "query={severity=\"warning\",file=\"/tmp/test.log\",host=\"$HOST\",tag=\"ci.integration.test\"}"

res=$(curl -sG "$LOKI_URL/loki/api/v1/query" --data-urlencode "query={severity=\"warning\",file=\"/tmp/test.log\",host=\"$HOST\",tag=\"ci.integration.test\"}")
[[ "$(echo $res | jq -r .data.result[0].values[0][1])" =~ "WARNING Something bad might happen" ]] || exit 1

res=$(curl -sG "$LOKI_URL/loki/api/v1/query" --data-urlencode "query={severity=\"critical\",file=\"/tmp/test.log\",host=\"$HOST\",tag=\"ci.integration.test\"}")
[[ "$(echo $res | jq -r .data.result[0].values[0][1])" =~ ":ERROR: Something bad happened" ]] || exit 1

res=$(curl -sG "$LOKI_URL/loki/api/v1/query" --data-urlencode "query={severity=\"debug\",file=\"/tmp/test.log\",host=\"$HOST\",tag=\"ci.integration.test\"}")
[[ "$(echo $res | jq -r .data.result[0].values[0][1])" =~ "[DEBUG] Wubba lubba dub dub" ]] || exit 1
