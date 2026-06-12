package main

import (
	"fmt"
	"os"
	"path/filepath"

	"go-chrome/internal/db"
	"go-chrome/internal/flow"
)

func main() {
	exe, _ := os.Executable()
	baseDir := filepath.Dir(exe)
	dataDir := filepath.Join(baseDir, "data")

	dbPath := filepath.Join(dataDir, "go-chrome.db")
	sqliteDB, err := db.Open(dbPath)
	if err != nil {
		fmt.Println("open db failed:", err)
		os.Exit(1)
	}
	defer sqliteDB.Close()

	store, err := db.NewFlowStore(sqliteDB)
	if err != nil {
		fmt.Println("create store failed:", err)
		os.Exit(1)
	}

	example := flow.NewExampleLoginFlow()
	if err := store.Save(example); err != nil {
		fmt.Println("save failed:", err)
		os.Exit(1)
	}

	fmt.Println("OK:", example.ID, example.Name)
}
