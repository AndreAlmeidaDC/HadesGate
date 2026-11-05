package detector

import (
	"strings"
	"unicode"
)

// ACMatch represents a pattern match found by the Aho-Corasick automaton.
type ACMatch struct {
	Pattern string
	Start   int // Byte offset start in original text
	End     int // Byte offset end in original text
}

type acNode struct {
	children map[rune]*acNode
	fail     *acNode
	output   []string
}

func newACNode() *acNode {
	return &acNode{
		children: make(map[rune]*acNode),
	}
}

// AhoCorasick implements a multi-pattern string searching automaton.
type AhoCorasick struct {
	root *acNode
}

// NewAhoCorasick compiles a set of pattern strings into an Aho-Corasick automaton.
func NewAhoCorasick(patterns []string) *AhoCorasick {
	root := newACNode()

	// 1. Build trie
	for _, p := range patterns {
		norm := normalizeMedicationText(p)
		if norm == "" {
			continue
		}
		curr := root
		for _, r := range norm {
			next, ok := curr.children[r]
			if !ok {
				next = newACNode()
				curr.children[r] = next
			}
			curr = next
		}
		curr.output = append(curr.output, p)
	}

	// 2. Build failure links via Breadth-First Search (BFS)
	queue := make([]*acNode, 0, 1024)

	for _, child := range root.children {
		child.fail = root
		queue = append(queue, child)
	}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for r, child := range curr.children {
			queue = append(queue, child)

			f := curr.fail
			for f != nil && f.children[r] == nil {
				f = f.fail
			}

			if f != nil {
				child.fail = f.children[r]
			} else {
				child.fail = root
			}

			if child.fail != nil && len(child.fail.output) > 0 {
				child.output = append(child.output, child.fail.output...)
			}
		}
	}

	return &AhoCorasick{root: root}
}

// Search scans text for matches with word-boundary awareness.
func (ac *AhoCorasick) Search(text string) []ACMatch {
	if ac == nil || ac.root == nil || text == "" {
		return nil
	}

	// Normalize text and map rune positions to original byte offsets
	type runePos struct {
		r         rune
		byteStart int
		byteEnd   int
	}

	runes := make([]runePos, 0, len(text))
	for byteIdx, r := range text {
		normR := foldRune(r)
		runes = append(runes, runePos{
			r:         normR,
			byteStart: byteIdx,
			byteEnd:   byteIdx + len(string(r)),
		})
	}

	curr := ac.root
	var matches []ACMatch

	for i := 0; i < len(runes); i++ {
		r := runes[i].r

		for curr != nil && curr.children[r] == nil {
			curr = curr.fail
		}

		if curr == nil {
			curr = ac.root
			continue
		}

		curr = curr.children[r]
		if curr == nil {
			curr = ac.root
			continue
		}

		if len(curr.output) > 0 {
			for _, pat := range curr.output {
				patRunes := len([]rune(normalizeMedicationText(pat)))
				startIdx := i - patRunes + 1
				if startIdx < 0 || startIdx >= len(runes) {
					continue
				}

				// Word boundary check: left side
				if startIdx > 0 {
					prevR := runes[startIdx-1].r
					if isWordRune(prevR) {
						continue
					}
				}

				// Word boundary check: right side
				if i+1 < len(runes) {
					nextR := runes[i+1].r
					if isWordRune(nextR) {
						continue
					}
				}

				byteStart := runes[startIdx].byteStart
				byteEnd := runes[i].byteEnd

				matches = append(matches, ACMatch{
					Pattern: pat,
					Start:   byteStart,
					End:     byteEnd,
				})
			}
		}
	}

	return matches
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func foldRune(r rune) rune {
	switch r {
	case 'á', 'à', 'ã', 'â', 'ä', 'Á', 'À', 'Ã', 'Â', 'Ä':
		return 'a'
	case 'é', 'è', 'ê', 'ë', 'É', 'È', 'Ê', 'Ë':
		return 'e'
	case 'í', 'ì', 'î', 'ï', 'Í', 'Ì', 'Î', 'Ï':
		return 'i'
	case 'ó', 'ò', 'õ', 'ô', 'ö', 'Ó', 'Ò', 'Õ', 'Ô', 'Ö':
		return 'o'
	case 'ú', 'ù', 'û', 'ü', 'Ú', 'Ù', 'Û', 'Ü':
		return 'u'
	case 'ç', 'Ç':
		return 'c'
	default:
		return unicode.ToLower(r)
	}
}

func normalizeMedicationText(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range s {
		folded := foldRune(r)
		if isWordRune(folded) || folded == ' ' || folded == '-' {
			sb.WriteRune(folded)
		}
	}
	return strings.Join(strings.Fields(sb.String()), " ")
}

