package main

import (
	"flag"
	"fmt"
	"log"
	"mail-indexer/config"
	"mail-indexer/indexer"
	"mail-indexer/parser"
	"mail-indexer/scanner"
	"os"
	"sync"
	"time"
)

func main() {
	account := flag.String("account", "", "cPanel account name")
	domain := flag.String("domain", "", "Email domain")
	user := flag.String("user", "", "Email user")
	actualUser := flag.String("actual-user", "", "Actual email owner (optional, defaults to --user)")
	before := flag.String("before", "2024-01-01", "Archive emails before this date (YYYY-MM-DD)")
	stats := flag.Bool("stats", false, "Show date statistics without indexing")
	delete := flag.Bool("delete", false, "Delete email after indexing succesfully")

	flag.Parse()

	if *account == "" || *domain == "" || *user == "" {
		log.Fatal("--account, --domain, and --user are required")
	}

	beforeDate, err := time.Parse("2006-01-02", *before)
	if err != nil {
		log.Fatalf("Invalid date format: %v", err)
	}

	startTime := time.Now()

	owner := *user
	if *actualUser != "" {
		owner = *actualUser
	}

	cfg := config.New(*user, *domain, *account, beforeDate)

	mailPath := cfg.GetMailPath()
	fmt.Printf("Scanning mailbox: %s\n", mailPath)

	scan := scanner.New(mailPath)
	parse := parser.New(owner+"@"+*domain, beforeDate)
	idx, err := indexer.New(cfg.ESHost, cfg.ESIndex)
	if err != nil {
		log.Fatalf("Failed to connect to Elasticsearch: %v", err)
	}

	// create index
	if err := idx.CreateIndex(); err != nil {
		log.Fatalf("Failed to create index: %v", err)
	}
	if err != nil {
		log.Fatalf("Failed to create index: %v", err)
	}

	//scan emails
	emailFiles, err := scan.ScanEmails()
	if err != nil {
		log.Fatalf("Failed to scan emails: %v", err)
	}
	fmt.Printf("Found %d email files\n", len(emailFiles))

	if *stats {
		beforeYear := 0
		afterYear := 0

		for i, filePath := range emailFiles {
			email, err := parse.ParseFile(filePath)
			if err != nil {
				continue
			}

			if email.Date.Before(beforeDate) {
				beforeYear++
			} else {
				afterYear++
			}

			if (i+1)%100 == 0 {
				fmt.Printf("Counted %d emails\n", i+1)
			}
		}

		fmt.Printf("\n\n//////////RESULT////////////\n\n")
		fmt.Printf("Before %s: %d\n", beforeDate.Format("2006-01-02"), beforeYear)
		fmt.Printf("After %s: %d\n", beforeDate.Format("2006-01-02"), afterYear)
		return
	}

	// process, concurrently
	indexed := 0
	skipped := 0
	failed := 0
	deleted := 0

	var mu sync.Mutex
	var wg sync.WaitGroup

	// Create channels
	numWorkers := 20
	fileChan := make(chan string, 100)

	for i := range numWorkers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for filePath := range fileChan {
				// parse email
				email, err := parse.ParseFile(filePath)
				if err != nil {
					mu.Lock()
					failed++
					mu.Unlock()
					log.Printf("Worker %d: Failed to parse %s: %v", workerID, filePath, err)
					continue
				}

				// check
				if !parse.ShouldIndex(email) {
					mu.Lock()
					skipped++
					mu.Unlock()
					continue
				}

				// index to elasticsearch
				if err := idx.IndexEmail(email); err != nil {
					mu.Lock()
					failed++
					mu.Unlock()
					log.Printf("Worker %d: Failed to index %s: %v", workerID, filePath, err)
					continue
				}

				mu.Lock()
				indexed++

				if *delete {
					if err := os.Remove(filePath); err != nil {
						log.Printf("Warning: Failed to delete %s: %v", filePath, err)
					} else {
						deleted++
					}
				}

				if indexed%100 == 0 {
					fmt.Printf("Indexed: %d, Skipped: %d, Failed: %d\n", indexed, skipped, failed)
				}
				mu.Unlock()
			}
		}(i)
	}

	// pass files to workers
	go func() {
		for _, filePath := range emailFiles {
			fileChan <- filePath
		}
		close(fileChan)
	}()

	// wait for all workers
	wg.Wait()

	elapsed := time.Since(startTime)

	finalMsg := fmt.Sprintf("\nDone! Indexed: %d, Skipped: %d, Failed: %d in %s\n",
		indexed, skipped, failed, elapsed.Round(time.Second))

	if *delete {
		finalMsg = fmt.Sprintf("\nDone! Indexed: %d, Skipped: %d, Failed: %d, Deleted %d in %s\n",
			indexed, skipped, failed, deleted, elapsed.Round(time.Second))
	}

	fmt.Printf(finalMsg)

}
