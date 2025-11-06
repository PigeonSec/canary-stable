package matcher

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"canary/internal/config"
	"canary/internal/models"

	ac "github.com/anknown/ahocorasick"
)

// Load loads keywords from a file and builds the Aho-Corasick automaton
func Load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var kws []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kws = append(kws, strings.ToLower(line))
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(kws) == 0 {
		return fmt.Errorf("no keywords found in %s", filename)
	}

	dict := make([][]rune, len(kws))
	for i, kw := range kws {
		dict[i] = []rune(kw)
	}

	var newMachine ac.Machine
	if err := newMachine.Build(dict); err != nil {
		return fmt.Errorf("build ACAutomaton: %w", err)
	}

	st := &models.MatcherState{
		Machine:  newMachine,
		Keywords: kws,
	}
	config.CurrentMatcher.Store(st)

	log.Printf("Reloaded %d keywords from %s", len(kws), filename)
	return nil
}

// GetCurrent retrieves the current matcher state
func GetCurrent() *models.MatcherState {
	v := config.CurrentMatcher.Load()
	if v == nil {
		return nil
	}
	return v.(*models.MatcherState)
}

// Find searches for keyword matches in the provided domains
func Find(domains []string) []string {
	st := GetCurrent()
	if st == nil {
		return nil
	}

	matchesMap := make(map[string]bool)

	for _, domain := range domains {
		if domain == "" {
			continue
		}
		lowered := strings.ToLower(domain)
		terms := st.Machine.MultiPatternSearch([]rune(lowered), false)
		for _, term := range terms {
			matchesMap[string(term.Word)] = true
		}
	}

	result := make([]string, 0, len(matchesMap))
	for k := range matchesMap {
		result = append(result, k)
	}
	return result
}

// AppendKeywords appends keywords to the keywords file
func AppendKeywords(path string, kws []string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && !os.IsExist(err) {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, kw := range kws {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		if _, err := f.WriteString(strings.ToLower(kw) + "\n"); err != nil {
			return err
		}
	}
	return nil
}
