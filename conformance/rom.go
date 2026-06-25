package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// ROMExpectation descrive una ROM locale senza includerne i byte.
type ROMExpectation struct {
	Name           string `json:"name"`
	ExpectedSize   int64  `json:"expected_size"`   // -1 ignora la dimensione
	ExpectedSHA256 string `json:"expected_sha256"` // vuoto ignora l'hash
}

// ROMVerification conserva dimensione/hash effettivi e l'esito.
type ROMVerification struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	ActualSize   int64  `json:"actual_size"`
	ActualSHA256 string `json:"actual_sha256"`
	Matches      bool   `json:"matches"`
}

// VerifyLocalROM calcola SHA-256 in streaming e confronta solo i vincoli dati.
func VerifyLocalROM(path string, expectation ROMExpectation) (ROMVerification, error) {
	result := ROMVerification{Name: expectation.Name, Path: path}
	if expectation.ExpectedSize < -1 {
		return result, fmt.Errorf("dimensione attesa non valida: %d", expectation.ExpectedSize)
	}
	expectedHash := strings.ToLower(strings.TrimSpace(expectation.ExpectedSHA256))
	if expectedHash != "" {
		decoded, err := hex.DecodeString(expectedHash)
		if err != nil || len(decoded) != sha256.Size {
			return result, fmt.Errorf("SHA-256 atteso non valido")
		}
	}

	file, err := os.Open(path)
	if err != nil {
		return result, err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return result, err
	}
	result.ActualSize = size
	result.ActualSHA256 = hex.EncodeToString(hash.Sum(nil))
	result.Matches = (expectation.ExpectedSize < 0 || expectation.ExpectedSize == size) &&
		(expectedHash == "" || expectedHash == result.ActualSHA256)
	return result, nil
}
