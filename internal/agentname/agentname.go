// Package agentname generates unique agent names for use with task claims.
package agentname

import (
	"bufio"
	"crypto/rand"
	"math/big"
	"os"
	"strings"
)

// dictPath is the system dictionary path, variable for testing.
var dictPath = "/usr/share/dict/words"

// nameWordCount is the number of words in a generated name.
const nameWordCount = 2

// Generate produces a name like "quiet-storm" by picking two random words.
// It tries the system dictionary first, then falls back to an embedded list.
func Generate() (string, error) {
	words := loadWords()

	parts := make([]string, nameWordCount)
	for i := range parts {
		idx, randErr := cryptoRandIntn(len(words))
		if randErr != nil {
			return "", randErr
		}
		parts[i] = words[idx]
	}

	return strings.Join(parts, "-"), nil
}

// loadWords returns a filtered word list. It prefers the system dictionary
// and falls back to the embedded list if unavailable.
func loadWords() []string {
	words, err := readDictFile()
	if err == nil && len(words) > 0 {
		return words
	}
	return embeddedWords()
}

// readDictFile reads and filters the system dictionary.
func readDictFile() ([]string, error) {
	f, err := os.Open(dictPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var words []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		w := scanner.Text()
		if isValidWord(w) {
			words = append(words, w)
		}
	}
	return words, scanner.Err()
}

// isValidWord checks that a word is 4-8 lowercase ASCII letters.
func isValidWord(w string) bool {
	n := len(w)
	const minLen, maxLen = 4, 8
	if n < minLen || n > maxLen {
		return false
	}
	for _, c := range w {
		if c < 'a' || c > 'z' {
			return false
		}
	}
	return true
}

// cryptoRandIntn returns a cryptographically random int in [0, n).
func cryptoRandIntn(n int) (int, error) {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, err
	}
	return int(v.Int64()), nil
}
