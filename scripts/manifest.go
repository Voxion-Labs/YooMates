package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type CommitInfo struct {
	Hash        string   `json:"hash"`
	ShortHash   string   `json:"short_hash"`
	AuthorName  string   `json:"author_name"`
	AuthorEmail string   `json:"author_email"`
	Timestamp   string   `json:"timestamp"`
	Subject     string   `json:"subject"`
	Files       []string `json:"files_changed"`
}

type Manifest struct {
	Project     string       `json:"project"`
	BotName     string       `json:"bot_name"`
	LastUpdated string       `json:"last_updated"`
	Branch      string       `json:"branch"`
	HeadCommit  string       `json:"head_commit"`
	History     []CommitInfo `json:"history"`
}

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func main() {
	branch, err := runGit("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		fmt.Printf("Warning: failed to get git branch: %v\n", err)
		branch = "unknown"
	}

	headCommit, err := runGit("rev-parse", "HEAD")
	if err != nil {
		fmt.Printf("Warning: failed to get git HEAD: %v\n", err)
		headCommit = "unknown"
	}

	// Fetch last 25 commits log
	logOut, err := runGit("log", "-n", "25", "--pretty=format:%H%x00%h%x00%an%x00%ae%x00%aI%x00%s")
	if err != nil {
		fmt.Printf("Error fetching git log: %v\n", err)
		os.Exit(1)
	}

	var history []CommitInfo
	lines := strings.Split(logOut, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\x00")
		if len(parts) < 6 {
			continue
		}

		hash := parts[0]
		shortHash := parts[1]
		authorName := parts[2]
		authorEmail := parts[3]
		timestamp := parts[4]
		subject := parts[5]

		// Get changed files for this commit
		filesOut, _ := runGit("diff-tree", "--no-commit-id", "--name-only", "-r", hash)
		var files []string
		for _, f := range strings.Split(filesOut, "\n") {
			f = strings.TrimSpace(f)
			if f != "" && f != "voxion-manifest.json" {
				files = append(files, f)
			}
		}

		history = append(history, CommitInfo{
			Hash:        hash,
			ShortHash:   shortHash,
			AuthorName:  authorName,
			AuthorEmail: authorEmail,
			Timestamp:   timestamp,
			Subject:     subject,
			Files:       files,
		})
	}

	manifest := Manifest{
		Project:     "YooMates",
		BotName:     "voxion-bot",
		LastUpdated: time.Now().UTC().Format(time.RFC3339),
		Branch:      branch,
		HeadCommit:  headCommit,
		History:     history,
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		fmt.Printf("Failed to marshal manifest JSON: %v\n", err)
		os.Exit(1)
	}

	manifestPath := "voxion-manifest.json"
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		fmt.Printf("Failed to write manifest file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[voxion-bot] Successfully updated %s with %d commits.\n", manifestPath, len(history))
}
