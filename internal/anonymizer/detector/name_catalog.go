package detector

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/binary"
	"io"
	"sync"
	"unicode"
)

//go:embed name_catalog.gz
var defaultNameCatalogGz []byte

const (
	NameTypeGiven   uint8 = 1
	NameTypeSurname uint8 = 2
	NameTypeBoth    uint8 = 3
)

// NameCatalogEntry stores classification and frequency data for a name token.
type NameCatalogEntry struct {
	Name        string
	Flags       uint8
	FreqGiven   uint32
	FreqSurname uint32
}

// nameACNode is a compact state in the Aho-Corasick automaton.
// Transitions are indexed by letter 'a'-'z' (0..25).
type nameACNode struct {
	next   [26]uint32
	fail   uint32
	nameID uint32 // 1-based index in catalog entries (0 = none)
}

// NameAhoCorasick implements an array-indexed Aho-Corasick automaton for fast candidate search.
type NameAhoCorasick struct {
	nodes   []nameACNode
	entries []NameCatalogEntry
}

var (
	globalNameAC   *NameAhoCorasick
	globalNameACOC sync.Once
)

// GetNameAhoCorasick returns the singleton pre-compiled Aho-Corasick automaton for person names.
func GetNameAhoCorasick() *NameAhoCorasick {
	globalNameACOC.Do(func() {
		globalNameAC = buildNameAhoCorasick(defaultNameCatalogGz)
	})
	return globalNameAC
}

func buildNameAhoCorasick(gzData []byte) *NameAhoCorasick {
	if len(gzData) == 0 {
		return &NameAhoCorasick{nodes: make([]nameACNode, 1)}
	}

	gz, err := gzip.NewReader(bytes.NewReader(gzData))
	if err != nil {
		return &NameAhoCorasick{nodes: make([]nameACNode, 1)}
	}
	defer gz.Close()

	magic := make([]byte, 5)
	if _, err := io.ReadFull(gz, magic); err != nil {
		return &NameAhoCorasick{nodes: make([]nameACNode, 1)}
	}

	var count uint32
	if err := binary.Read(gz, binary.LittleEndian, &count); err != nil {
		return &NameAhoCorasick{nodes: make([]nameACNode, 1)}
	}

	entries := make([]NameCatalogEntry, count)
	buf := make([]byte, 32)

	for i := uint32(0); i < count; i++ {
		var header [2]byte
		if _, err := io.ReadFull(gz, header[:]); err != nil {
			break
		}
		l := header[0]
		fl := header[1]
		nameBuf := buf[:l]
		if _, err := io.ReadFull(gz, nameBuf); err != nil {
			break
		}
		var freqs [2]uint32
		if err := binary.Read(gz, binary.LittleEndian, &freqs); err != nil {
			break
		}
		entries[i] = NameCatalogEntry{
			Name:        string(nameBuf),
			Flags:       fl,
			FreqGiven:   freqs[0],
			FreqSurname: freqs[1],
		}
	}

	// 1. Build Trie
	nodes := make([]nameACNode, 1, 600000) // index 0 is root

	for id, entry := range entries {
		curr := uint32(0)
		for i := 0; i < len(entry.Name); i++ {
			c := entry.Name[i] - 'a'
			if c >= 26 {
				continue
			}
			nxt := nodes[curr].next[c]
			if nxt == 0 {
				nxt = uint32(len(nodes))
				nodes = append(nodes, nameACNode{})
				nodes[curr].next[c] = nxt
			}
			curr = nxt
		}
		nodes[curr].nameID = uint32(id + 1)
	}

	// 2. Build failure links via BFS
	queue := make([]uint32, 0, 1024)
	for c := 0; c < 26; c++ {
		child := nodes[0].next[c]
		if child != 0 {
			nodes[child].fail = 0
			queue = append(queue, child)
		}
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for c := 0; c < 26; c++ {
			child := nodes[curr].next[c]
			if child != 0 {
				queue = append(queue, child)
				f := nodes[curr].fail
				for f != 0 && nodes[f].next[c] == 0 {
					f = nodes[f].fail
				}
				nodes[child].fail = nodes[f].next[c]
			}
		}
	}

	return &NameAhoCorasick{
		nodes:   nodes,
		entries: entries,
	}
}

// Lookup queries the trie in O(len) for exact name classification and frequency.
func (ac *NameAhoCorasick) Lookup(norm string) (NameCatalogEntry, bool) {
	if ac == nil || len(ac.nodes) == 0 {
		return NameCatalogEntry{}, false
	}
	curr := uint32(0)
	for i := 0; i < len(norm); i++ {
		b := norm[i]
		if b < 'a' || b > 'z' {
			return NameCatalogEntry{}, false
		}
		c := b - 'a'
		curr = ac.nodes[curr].next[c]
		if curr == 0 {
			return NameCatalogEntry{}, false
		}
	}
	id := ac.nodes[curr].nameID
	if id == 0 || int(id-1) >= len(ac.entries) {
		return NameCatalogEntry{}, false
	}
	return ac.entries[id-1], true
}

// NameCandidateToken represents an isolated candidate token identified in the text.
type NameCandidateToken struct {
	Text        string
	Norm        string
	Start       int
	End         int
	Flags       uint8
	FreqGiven   uint32
	FreqSurname uint32
	IsTitleCase bool
	IsAllCaps   bool
}

// SearchCandidateTokens runs Aho-Corasick across text, filtering for word-boundary matches.
func (ac *NameAhoCorasick) SearchCandidateTokens(text string) []NameCandidateToken {
	if ac == nil || len(ac.nodes) == 0 || text == "" {
		return nil
	}

	type runePos struct {
		r         rune
		c         byte // 0..25 if 'a'..'z', 255 otherwise
		byteStart int
		byteEnd   int
	}

	runes := make([]runePos, 0, len(text))
	for byteIdx, r := range text {
		folded := foldRune(r)
		var c byte = 255
		if folded >= 'a' && folded <= 'z' {
			c = byte(folded - 'a')
		}
		runes = append(runes, runePos{
			r:         r,
			c:         c,
			byteStart: byteIdx,
			byteEnd:   byteIdx + len(string(r)),
		})
	}

	var candidates []NameCandidateToken
	curr := uint32(0)

	for i := 0; i < len(runes); i++ {
		c := runes[i].c
		if c == 255 {
			curr = 0
			continue
		}

		for curr != 0 && ac.nodes[curr].next[c] == 0 {
			curr = ac.nodes[curr].fail
		}
		curr = ac.nodes[curr].next[c]

		if ac.nodes[curr].nameID != 0 {
			entry := ac.entries[ac.nodes[curr].nameID-1]
			patLen := len(entry.Name)
			startRuneIdx := i - patLen + 1
			if startRuneIdx < 0 || startRuneIdx >= len(runes) {
				continue
			}

			// Left word boundary check: character before start must not be a letter or digit
			if startRuneIdx > 0 {
				prevR := runes[startRuneIdx-1].r
				if unicode.IsLetter(prevR) || unicode.IsDigit(prevR) {
					continue
				}
			}

			// Right word boundary check: character after end must not be a letter or digit
			if i+1 < len(runes) {
				nextR := runes[i+1].r
				if unicode.IsLetter(nextR) || unicode.IsDigit(nextR) {
					continue
				}
			}

			byteStart := runes[startRuneIdx].byteStart
			byteEnd := runes[i].byteEnd
			rawWord := text[byteStart:byteEnd]

			isTitle := isTitleCaseWord(rawWord)
			isUpper := isAllUpperToken(rawWord)

			candidates = append(candidates, NameCandidateToken{
				Text:        rawWord,
				Norm:        entry.Name,
				Start:       byteStart,
				End:         byteEnd,
				Flags:       entry.Flags,
				FreqGiven:   entry.FreqGiven,
				FreqSurname: entry.FreqSurname,
				IsTitleCase: isTitle,
				IsAllCaps:   isUpper,
			})
		}
	}

	return candidates
}

func isTitleCaseWord(word string) bool {
	runes := []rune(word)
	if len(runes) == 0 {
		return false
	}
	if !unicode.IsUpper(runes[0]) {
		return false
	}
	for i := 1; i < len(runes); i++ {
		if !unicode.IsLower(runes[i]) && runes[i] != '-' && runes[i] != '\'' {
			return false
		}
	}
	return true
}

