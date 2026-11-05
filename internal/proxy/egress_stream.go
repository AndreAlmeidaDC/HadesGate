package proxy

import (
	"strings"
	"unicode/utf8"

	"github.com/hadesgate/hadesgate/internal/anonymizer"
)

const (
	egressStreamLookbehind = 256
	egressStreamHardBuffer = 4096
)

// egressStreamFilter intentionally retains a bounded suffix so detectors can
// inspect sensitive values that providers split across multiple SSE events.
type egressStreamFilter struct {
	engine          *anonymizer.Engine
	knownPlaintexts map[string]bool
	pending         string
	onDetected      func([]anonymizer.DetectedEntity)
}

func (f *egressStreamFilter) Feed(fragment string) string {
	f.pending += fragment
	if len(f.pending) <= egressStreamLookbehind {
		return ""
	}

	cutLimit := len(f.pending) - egressStreamLookbehind
	cut := strings.LastIndexAny(f.pending[:cutLimit], " \t\r\n,.;:!?")
	if cut >= 0 {
		cut++
	} else if len(f.pending) > egressStreamHardBuffer {
		cut = cutLimit
		for cut > 0 && !utf8.RuneStart(f.pending[cut]) {
			cut--
		}
	} else {
		return ""
	}

	return f.scanPrefix(cut)
}

func (f *egressStreamFilter) Flush() string {
	return f.scanPrefix(len(f.pending))
}

func (f *egressStreamFilter) scanPrefix(cut int) string {
	if cut <= 0 {
		return ""
	}
	text := f.pending[:cut]
	f.pending = f.pending[cut:]
	result := f.engine.ScanEgress(f.knownPlaintexts, text)
	if len(result.LeakedEntities) > 0 && f.onDetected != nil {
		f.onDetected(result.LeakedEntities)
	}
	return result.SanitizedText
}
