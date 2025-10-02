package indexer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mail-indexer/parser"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

type Indexer struct {
	client *elasticsearch.Client
	index  string
}

func New(esHost, index string) (*Indexer, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{esHost},
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Indexer{
		client: client,
		index:  index,
	}, nil
}

func (i *Indexer) CreateIndex() error {
	exists, err := i.client.Indices.Exists([]string{i.index})
	if err != nil {
		return err
	}

	if exists.StatusCode == 200 {
		return nil // Index already exists
	}

	// Create index with mapping
	mapping := `{
        "mappings": {
            "properties": {
                "message_id": {"type": "keyword"},
                "user": {"type": "keyword"},
                "subject": {"type": "text"},
                "from": {"type": "keyword"},
                "to": {"type": "keyword"},
                "date": {"type": "date"},
                "body": {"type": "text"},
                "attachments": {
                    "properties": {
                        "filename": {"type": "text"},
                        "content_type": {"type": "keyword"},
                        "size": {"type": "integer"}
                    }
                }
            }
        }
    }`

	req := esapi.IndicesCreateRequest{
		Index: i.index,
		Body:  bytes.NewReader([]byte(mapping)),
	}

	res, err := req.Do(context.Background(), i.client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to create index: %s", res.String())
	}

	return nil

}

func (i *Indexer) IndexEmail(email *parser.Email) error {
	doc := map[string]any{
		"message_id": email.MessageID,
		"user":       email.User,
		"subject":    email.Subject,
		"from":       email.From,
		"to":         email.To,
		"date":       email.Date,
		"body":       email.Body,
	}

	// attachments metadata
	var attachments []map[string]any
	for _, att := range email.Attachments {
		attachments = append(attachments, map[string]any{
			"filename":     att.Filename,
			"content_type": att.ContentType,
			"size":         att.Size,
		})
	}

	doc["attachments"] = attachments

	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	hash := sha256.Sum256([]byte(email.MessageID))
	docID := hex.EncodeToString(hash[:])

	req := esapi.IndexRequest{
		Index:      i.index,
		DocumentID: docID,
		Body:       bytes.NewReader(body),
		Refresh:    "false",
	}

	res, err := req.Do(context.Background(), i.client)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("failed to index email: %s", res.String())
	}

	return nil

}
