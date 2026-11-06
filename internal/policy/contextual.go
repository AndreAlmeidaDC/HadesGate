package policy

import (
	"strings"

	"github.com/hadesgate/hadesgate/internal/anonymizer/detector"
)

const maxContextualLinkDistance = 240

var identityAnchorTypes = map[detector.EntityType]bool{
	detector.EntityName: true, detector.EntityCPF: true, detector.EntityRG: true,
	detector.EntityCNH: true, detector.EntityEmail: true, detector.EntityPhone: true,
	detector.EntitySUS: true, detector.EntityMedicalID: true, detector.EntityPassport: true,
	detector.EntitySSN: true, detector.EntityNINO: true, detector.EntityNHS: true,
	detector.EntityDNI: true, detector.EntityNIR: true, detector.EntityCodiceFiscale: true,
	detector.EntityGovernmentID: true,
}

// FilterContextualMatches keeps sensitive attributes only when they are linked
// to a direct identity anchor in the same short textual segment. Explicit mask,
// block, route_local, or disabled configuration continues to take precedence.
func (e *Engine) FilterContextualMatches(text string, matches []detector.Match) []detector.Match {
	filtered := make([]detector.Match, 0, len(matches))
	for _, match := range matches {
		if e.ActionFor(string(match.Type)) != Contextual || linkedToIdentity(text, match, matches) {
			filtered = append(filtered, match)
		}
	}
	return filtered
}

func linkedToIdentity(text string, attribute detector.Match, matches []detector.Match) bool {
	for _, candidate := range matches {
		if !identityAnchorTypes[candidate.Type] || rangesOverlap(attribute, candidate) {
			continue
		}
		left, right := attribute, candidate
		if candidate.Start < attribute.Start {
			left, right = candidate, attribute
		}
		if left.End > right.Start || right.Start-left.End > maxContextualLinkDistance {
			continue
		}
		between := text[left.End:right.Start]
		if !strings.ContainsAny(between, ".!?;\n\r") {
			return true
		}
	}
	return false
}

func rangesOverlap(left, right detector.Match) bool {
	return left.Start < right.End && right.Start < left.End
}
