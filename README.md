# mail-indexer


ex: ./mail-indexer --account westgate --domain westgate.ng --user duke --before 2024-01-01
--stats // for just stats
--delete // to delete the email after indexing


# Count documents
curl http://localhost:9200/mail-archive/_count

# See a sample email
curl http://localhost:9200/mail-archive/_search?size=1&pretty

# Search for emails containing "invoice"
curl -X GET "localhost:9200/mail-archive/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "query": {
    "match": {
      "body": "invoice"
    }
  }
}'


# Delete the index (wipes all data)
curl -X DELETE "localhost:9200/mail-archive"
