package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"mail-indexer/config"
	"mail-indexer/indexer"
	"mail-indexer/parser"
	"mail-indexer/scanner"
	"os"
	"strings"
	"sync"
	"time"
)

func ask(question string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(question)
	answer, _ := reader.ReadString('\n')
	return strings.TrimSpace(answer)
}

func main() {
	account := flag.String("account", "", "cPanel account name")
	domain := flag.String("domain", "", "Email domain")
	user := flag.String("user", "", "Email user")
	actualUser := flag.String("actual-user", "", "Actual email owner (optional, defaults to --user)")
	before := flag.String("before", "2024-01-01", "Archive emails before this date (YYYY-MM-DD)")
	stats := flag.Bool("stats", false, "Show date statistics without indexing")
	// delete := flag.Bool("delete", false, "Delete email after indexing succesfully")

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

	userAction := "index"
	if *stats {
		userAction = "check stats for"
	}
	for {
		proceedToIndex := ask(fmt.Sprintf("\nDo you want to proceed to %s %d emails? (y/n)\n: ", userAction, len(emailFiles)))

		if proceedToIndex == "y" {
			break
		} else if proceedToIndex == "n" {
			fmt.Println("---exit---")
			return
		}
		// invalid input, loop continues and asks again
		fmt.Println("Please answer y or n")
	}

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
				if indexed%100 == 0 {
					fmt.Printf("Indexed: %d, Skipped: %d, Failed: %d, Completed: %d%% \n", indexed, skipped, failed, (indexed+skipped+failed)/len(emailFiles))
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

	fmt.Printf("\nDone! Indexed: %d, Skipped: %d, Failed: %d in %s\n",
		indexed, skipped, failed, elapsed.Round(time.Second))

	for {
		proceedToDelete := ask("\n Do you want to delete the indexed files?")
		if proceedToDelete == "y" {
			break
		} else if proceedToDelete == "n" {
			fmt.Println("---exit---")
			return
		}

		fmt.Println("Please answer y or n")
	}

	deleteFileChan := make(chan string, 100)
	var deleteWg sync.WaitGroup
	deleted := 0

	for i := range numWorkers {
		deleteWg.Add(1)
		go func(workerID int) {
			defer deleteWg.Done()
			for filePath := range deleteFileChan {
				if err := os.Remove(filePath); err != nil {
					log.Printf("Worker %d: Failed to delete %s: %v", workerID, filePath, err)
				} else {
					mu.Lock()
					deleted++
					mu.Unlock()
				}
			}
		}(i)
	}

	go func() {
		for _, filePath := range emailFiles {
			deleteFileChan <- filePath
		}
		close(deleteFileChan)
	}()

	fmt.Printf("Done deleting %d emails\n", deleted)
	fmt.Println("Thanks for using Mail-Indexer by sevena")
	fmt.Println("---exit---")
}
