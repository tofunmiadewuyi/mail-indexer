# mail-indexer


ex: ./mail-indexer --account westgate --domain westgate.ng --user duke --before 2024-01-01
--stats // for just stats
--delete // to delete the email after indexing


## Count documents
```curl http://localhost:9200/index_name/_count```

## See a sample email
```curl http://localhost:9200/index_name/_search?size=1&pretty```

## Search for emails containing "invoice"
```
curl -X GET "localhost:9200/index_name/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "query": {
    "match": {
      "body": "invoice"
    }
  }
}'
```

## Count emails for a user
```
curl -X GET "localhost:9200/index_name/_count?pretty" -H 'Content-Type: application/json' -d'
{
  "query": {
    "term": {
      "user": "username@domain.com"
    }
  }
}
'
```

## Get index size for that user
```
curl -X GET "localhost:9200/index_name/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "size": 0,
  "query": {
    "term": {
      "user": "username@domain.com"
    }
  },
  "aggs": {
    "total_size": {
      "sum": {
        "field": "_size"
      }
    }
  }
}
'
```


# Delete the index (wipes all data)
```curl -X DELETE "localhost:9200/index_name"```
