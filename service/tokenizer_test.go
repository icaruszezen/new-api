package service

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tiktoken-go/tokenizer"
	"github.com/tiktoken-go/tokenizer/codec"
)

// Long runs without separators are cut before encoding so BPE stays linear;
// ordinary text is counted exactly and cut runs stay within 0.5%.
func TestGetTokenNumCutsOnlyLongRuns(t *testing.T) {
	ordinary := strings.Repeat("func main() {\n\tfmt.Println(\"统计 token，按模型计费。\", 42)\n}\n| a | b |\n|---|---|\n", 200)
	rng := rand.New(rand.NewSource(1))
	var cjk strings.Builder
	for range 5000 {
		cjk.WriteRune(rune(0x4E00 + rng.Intn(0x9FA5-0x4E00)))
	}
	longRuns := []string{cjk.String(), strings.Repeat("token", 2000), "a" + strings.Repeat(" ", 10000) + "b", strings.Repeat("=", 10000)}

	for name, encoder := range map[string]tokenizer.Codec{"o200k_base": codec.NewO200kBase(), "cl100k_base": codec.NewCl100kBase()} {
		t.Run(name, func(t *testing.T) {
			want, _ := encoder.Count(ordinary)
			assert.Equal(t, want, getTokenNum(encoder, ordinary))

			for _, text := range longRuns {
				exact, _ := encoder.Count(text)
				assert.InEpsilon(t, exact, getTokenNum(encoder, text), 0.005)
			}
		})
	}
}
