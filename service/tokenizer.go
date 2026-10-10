package service

import (
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/tiktoken-go/tokenizer"
	"github.com/tiktoken-go/tokenizer/codec"
)

// tokenEncoderMap won't grow after initialization
var defaultTokenEncoder tokenizer.Codec

// tokenEncoderMap is used to store token encoders for different models
var tokenEncoderMap = make(map[string]tokenizer.Codec)

// tokenEncoderMutex protects tokenEncoderMap for concurrent access
var tokenEncoderMutex sync.RWMutex

func InitTokenEncoders() {
	common.SysLog("initializing token encoders")
	defaultTokenEncoder = codec.NewCl100kBase()
	common.SysLog("token encoders initialized")
}

func getTokenEncoder(model string) tokenizer.Codec {
	// First, try to get the encoder from cache with read lock
	tokenEncoderMutex.RLock()
	if encoder, exists := tokenEncoderMap[model]; exists {
		tokenEncoderMutex.RUnlock()
		return encoder
	}
	tokenEncoderMutex.RUnlock()

	// If not in cache, create new encoder with write lock
	tokenEncoderMutex.Lock()
	defer tokenEncoderMutex.Unlock()

	// Double-check if another goroutine already created the encoder
	if encoder, exists := tokenEncoderMap[model]; exists {
		return encoder
	}

	// Create new encoder
	modelCodec, err := tokenizer.ForModel(tokenizer.Model(model))
	if err != nil {
		// Cache the default encoder for this model to avoid repeated failures
		tokenEncoderMap[model] = defaultTokenEncoder
		return defaultTokenEncoder
	}

	// Cache the new encoder
	tokenEncoderMap[model] = modelCodec
	return modelCodec
}

// maxTokenRunBytes bounds a run of letters, whitespace, or symbols passed to
// the encoder in one piece. BPE cost grows with the square of a pre-tokenized
// piece: 100,000 Chinese characters without punctuation take tens of seconds
// in one piece. Cutting longer runs keeps counting linear; ordinary text has
// no such runs, and a cut changes the count only inside the run (under 0.5%
// for CJK).
const maxTokenRunBytes = 1024

func getTokenNum(tokenEncoder tokenizer.Codec, text string) int {
	if text == "" {
		return 0
	}
	total, start := 0, 0
	runClass, runStart := tokenRunNone, 0
	for i, r := range text {
		class := tokenRunClassOf(r)
		if class != runClass {
			runClass, runStart = class, i
			continue
		}
		if class != tokenRunNumber && i-runStart >= maxTokenRunBytes {
			n, _ := tokenEncoder.Count(text[start:i])
			total += n
			start, runStart = i, i
		}
	}
	n, _ := tokenEncoder.Count(text[start:])
	return total + n
}

type tokenRunClass int

const (
	tokenRunNone tokenRunClass = iota
	tokenRunLetter
	tokenRunNumber
	tokenRunSpace
	tokenRunSymbol
)

// tokenRunClassOf groups runes the way tiktoken's pre-tokenizer joins them
// into pieces. Digits are split into groups of three by the encoder itself.
func tokenRunClassOf(r rune) tokenRunClass {
	if r < utf8.RuneSelf {
		switch {
		case 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z':
			return tokenRunLetter
		case '0' <= r && r <= '9':
			return tokenRunNumber
		case r == ' ' || '\t' <= r && r <= '\r':
			return tokenRunSpace
		default:
			return tokenRunSymbol
		}
	}
	switch {
	case unicode.IsLetter(r) || unicode.IsMark(r):
		return tokenRunLetter
	case unicode.IsNumber(r):
		return tokenRunNumber
	case unicode.IsSpace(r):
		return tokenRunSpace
	default:
		return tokenRunSymbol
	}
}
