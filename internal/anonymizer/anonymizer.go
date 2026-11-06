package anonymizer

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/hadesgate/hadesgate/internal/anonymizer/detector"
	"github.com/hadesgate/hadesgate/internal/anonymizer/vault"
)

// DetectedEntity describes an entity found and masked in the text.
type DetectedEntity struct {
	Type         detector.EntityType `json:"type"`
	SubType      string              `json:"sub_type,omitempty"`
	Token        string              `json:"token"`
	Original     string              `json:"original"`
	Start        int                 `json:"start"`
	End          int                 `json:"end"`
	Score        float64             `json:"score,omitempty"`
	ContextFound []string            `json:"context_found,omitempty"`
}

// Result contains the output of the anonymization process.
type Result struct {
	SanitizedText string            `json:"sanitized_text"`
	Entities      []DetectedEntity  `json:"entities"`
	TokenMap      map[string]string `json:"token_map"`
}

// AnonymizeOptions customizes detection for a single request without mutating the engine.
type AnonymizeOptions struct {
	// IsTypeDisabled is evaluated before tokenization. Disabled matches remain
	// temporarily visible to relationship filters, then pass through unchanged
	// and are excluded from metrics and the vault.
	IsTypeDisabled func(detector.EntityType) bool
	// FilterMatches can apply request-scoped relationship policies after all
	// detectors and confidence thresholds run, but before tokenization.
	FilterMatches func(text string, matches []detector.Match) []detector.Match
}

// Engine coordinates PII detectors and token vault management.
type Engine struct {
	detectors        []detector.Detector
	vault            vault.Vault
	ttl              time.Duration
	minConfidence    float64
	familyThresholds map[string]float64
	entityThresholds map[string]float64
	thresholdsMu     sync.RWMutex
	allowlist        map[string]bool
	denylist         []string
	denylistRegexes  []*regexp.Regexp
}

// Config defines configuration for the anonymization engine.
type Config struct {
	EnableCPF           bool
	EnableCNPJ          bool
	EnablePhone         bool
	EnableEmail         bool
	EnableCreditCard    bool
	EnableName          bool
	EnableCEP           bool
	EnablePIX           bool
	EnableRG            bool
	EnableCNH           bool
	EnableBank          bool
	EnableIBAN          bool
	EnablePassport      bool
	EnableIPAddress     bool
	EnableAPIKey        bool
	EnableSecret        bool
	EnableSSN           bool
	EnableEIN           bool
	EnableNINO          bool
	EnableNHS           bool
	EnableDNI           bool
	EnableNIR           bool
	EnableCodiceFiscale bool
	EnableMedicalID     bool
	EnableSUS           bool
	EnableVehicle       bool
	EnableLegal         bool
	EnableDevice        bool
	EnableGovernmentID  bool
	EnableCID           bool
	EnableMedication    bool
	MinConfidence       float64
	FamilyThresholds    map[string]float64
	EntityThresholds    map[string]float64
	Allowlist           []string
	Denylist            []string
	CustomDetectorsFile string
	AdditionalDetectors []detector.Detector
	SessionTTL          time.Duration
}

// DefaultConfig returns the default configuration with all Brazilian LGPD, international and secret detectors enabled.
func DefaultConfig() Config {
	return Config{
		EnableCPF:           true,
		EnableCNPJ:          true,
		EnablePhone:         true,
		EnableEmail:         true,
		EnableCreditCard:    true,
		EnableName:          true,
		EnableCEP:           true,
		EnablePIX:           true,
		EnableRG:            true,
		EnableCNH:           true,
		EnableBank:          true,
		EnableIBAN:          true,
		EnablePassport:      true,
		EnableIPAddress:     true,
		EnableAPIKey:        true,
		EnableSecret:        true,
		EnableSSN:           true,
		EnableEIN:           true,
		EnableNINO:          true,
		EnableNHS:           true,
		EnableDNI:           true,
		EnableNIR:           true,
		EnableCodiceFiscale: true,
		EnableMedicalID:     true,
		EnableSUS:           true,
		EnableVehicle:       true,
		EnableLegal:         true,
		EnableDevice:        true,
		EnableGovernmentID:  true,
		EnableCID:           true,
		EnableMedication:    true,
		MinConfidence:       0.70,
		SessionTTL:          10 * time.Minute,
	}
}

// NewEngine creates a new Anonymizer Engine with configured detectors and vault.
func NewEngine(cfg Config, v vault.Vault) *Engine {
	var detectors []detector.Detector

	if cfg.EnableCPF {
		detectors = append(detectors, detector.NewCPFDetector())
	}
	if cfg.EnableCNPJ {
		detectors = append(detectors, detector.NewCNPJDetector())
	}
	if cfg.EnablePhone {
		detectors = append(detectors, detector.NewPhoneDetector())
	}
	if cfg.EnableEmail {
		detectors = append(detectors, detector.NewEmailDetector())
	}
	if cfg.EnableCreditCard {
		detectors = append(detectors, detector.NewCreditCardDetector())
	}
	if cfg.EnableName {
		detectors = append(detectors, detector.NewNameDetector())
	}
	if cfg.EnableCEP {
		detectors = append(detectors, detector.NewCEPDetector())
	}
	if cfg.EnablePIX {
		detectors = append(detectors, detector.NewPIXDetector())
	}
	if cfg.EnableRG {
		detectors = append(detectors, detector.NewRGDetector())
	}
	if cfg.EnableCNH {
		detectors = append(detectors, detector.NewCNHDetector())
	}
	if cfg.EnableBank {
		detectors = append(detectors, detector.NewBankAccountDetector())
	}
	if cfg.EnableIBAN {
		detectors = append(detectors, detector.NewIBANDetector())
	}
	if cfg.EnablePassport {
		detectors = append(detectors, detector.NewPassportDetector())
	}
	if cfg.EnableIPAddress {
		detectors = append(detectors, detector.NewIPDetector())
	}
	if cfg.EnableAPIKey {
		detectors = append(detectors, detector.NewAPIKeyDetector())
	}
	if cfg.EnableSecret {
		detectors = append(detectors, detector.NewSecretDetector())
	}
	if cfg.EnableSSN {
		detectors = append(detectors, detector.NewSSNDetector())
	}
	if cfg.EnableEIN {
		detectors = append(detectors, detector.NewEINDetector())
	}
	if cfg.EnableNINO {
		detectors = append(detectors, detector.NewNINODetector())
	}
	if cfg.EnableNHS {
		detectors = append(detectors, detector.NewNHSDetector())
	}
	if cfg.EnableDNI {
		detectors = append(detectors, detector.NewDNIDetector())
	}
	if cfg.EnableNIR {
		detectors = append(detectors, detector.NewNIRDetector())
	}
	if cfg.EnableCodiceFiscale {
		detectors = append(detectors, detector.NewCodiceFiscaleDetector())
	}
	if cfg.EnableMedicalID {
		detectors = append(detectors, detector.NewMedicalDetector())
	}
	if cfg.EnableSUS {
		detectors = append(detectors, detector.NewSUSDetector())
	}
	if cfg.EnableVehicle {
		detectors = append(detectors, detector.NewVehicleDetector())
	}
	if cfg.EnableLegal {
		detectors = append(detectors, detector.NewLegalDetector())
	}
	if cfg.EnableDevice {
		detectors = append(detectors, detector.NewDeviceDetector())
	}
	if cfg.EnableGovernmentID {
		detectors = append(detectors, detector.NewGovernmentIDDetector())
	}
	if cfg.EnableCID {
		detectors = append(detectors, detector.NewCIDDetector())
	}
	if cfg.EnableMedication {
		detectors = append(detectors, detector.NewMedicationDetector())
	}
	if cfg.CustomDetectorsFile != "" {
		rules, err := detector.LoadCustomRulesFromFile(cfg.CustomDetectorsFile)
		if err == nil && len(rules) > 0 {
			detectors = append(detectors, detector.NewCustomDetector(rules))
		}
	}
	detectors = append(detectors, cfg.AdditionalDetectors...)

	minConf := cfg.MinConfidence
	if minConf <= 0 {
		minConf = 0.70
	}

	allowMap := make(map[string]bool, len(cfg.Allowlist))
	for _, item := range cfg.Allowlist {
		trimmed := strings.ToLower(strings.TrimSpace(item))
		if trimmed != "" {
			allowMap[trimmed] = true
		}
	}

	cleanDenylist := make([]string, 0, len(cfg.Denylist))
	denylistRegexes := make([]*regexp.Regexp, 0, len(cfg.Denylist))
	for _, item := range cfg.Denylist {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			if rx, err := regexp.Compile("(?i)" + regexp.QuoteMeta(trimmed)); err == nil {
				cleanDenylist = append(cleanDenylist, trimmed)
				denylistRegexes = append(denylistRegexes, rx)
			}
		}
	}

	return &Engine{
		detectors:        detectors,
		vault:            v,
		ttl:              cfg.SessionTTL,
		minConfidence:    minConf,
		familyThresholds: cfg.FamilyThresholds,
		entityThresholds: cfg.EntityThresholds,
		allowlist:        allowMap,
		denylist:         cleanDenylist,
		denylistRegexes:  denylistRegexes,
	}
}

// SetThresholds dynamically updates family and entity confidence thresholds in RAM with atomic locking.
func (e *Engine) SetThresholds(family map[string]float64, entity map[string]float64) {
	e.thresholdsMu.Lock()
	defer e.thresholdsMu.Unlock()
	e.familyThresholds = cloneThresholds(family)
	e.entityThresholds = cloneThresholds(entity)
}

func cloneThresholds(source map[string]float64) map[string]float64 {
	cloned := make(map[string]float64, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

// ThresholdFor returns the effective confidence score threshold for a given entity type.
func (e *Engine) ThresholdFor(entityType detector.EntityType) float64 {
	e.thresholdsMu.RLock()
	defer e.thresholdsMu.RUnlock()
	return ResolveConfidenceThreshold(entityType, e.minConfidence, e.familyThresholds, e.entityThresholds)
}

// Vault returns the underlying token vault.
func (e *Engine) Vault() vault.Vault {
	return e.vault
}

// MedicationDetector returns the active MedicationDetector instance if registered.
func (e *Engine) MedicationDetector() *detector.MedicationDetector {
	for _, d := range e.detectors {
		if med, ok := d.(*detector.MedicationDetector); ok {
			return med
		}
	}
	return nil
}

// Anonymize scans the text for sensitive information, replaces it with surrogate tokens,
// stores the token mapping in the vault, and returns the sanitized text.
func (e *Engine) Anonymize(sessionID, text string) Result {
	return e.AnonymizeWithOptions(sessionID, text, AnonymizeOptions{})
}

// AnonymizeWithOptions scans and anonymizes text using request-scoped policy options.
func (e *Engine) AnonymizeWithOptions(sessionID, text string, options AnonymizeOptions) Result {
	if text == "" {
		return Result{SanitizedText: "", Entities: nil, TokenMap: nil}
	}

	var allMatches []detector.Match

	// 1. Scan Denylist terms first with maximum priority (UTF-8 byte-safe regex indices)
	for _, rx := range e.denylistRegexes {
		if options.IsTypeDisabled != nil && options.IsTypeDisabled(detector.EntityType("CONFIDENTIAL")) {
			break
		}
		indices := rx.FindAllStringIndex(text, -1)
		for _, idx := range indices {
			allMatches = append(allMatches, detector.Match{
				Type:            detector.EntityType("CONFIDENTIAL"),
				Value:           text[idx[0]:idx[1]],
				Start:           idx[0],
				End:             idx[1],
				Score:           1.0,
				ValidatorPassed: true,
			})
		}
	}

	// 2. Collect matches from all active detectors
	for _, d := range e.detectors {
		matches := d.Detect(text)
		for _, m := range matches {
			// Apply dynamic Confidence Score filter (per entity or family)
			if m.Score < e.ThresholdFor(m.Type) {
				continue
			}
			// Apply Allowlist filter
			if e.allowlist[strings.ToLower(m.Value)] {
				continue
			}
			allMatches = append(allMatches, m)
		}
	}
	if options.FilterMatches != nil && len(allMatches) > 0 {
		allMatches = options.FilterMatches(text, allMatches)
	}
	if options.IsTypeDisabled != nil && len(allMatches) > 0 {
		enabledMatches := allMatches[:0]
		for _, match := range allMatches {
			if !options.IsTypeDisabled(match.Type) {
				enabledMatches = append(enabledMatches, match)
			}
		}
		allMatches = enabledMatches
	}

	if len(allMatches) == 0 {
		return Result{
			SanitizedText: text,
			Entities:      nil,
			TokenMap:      nil,
		}
	}

	// 3. Intelligent Disambiguation and Overlap Resolution (PasteGuard merge.py parity):
	// Validated deterministic detectors always take precedence over semantic NER predictions.
	// Among same-tier matches, highest Score wins; on tie, longest match wins.
	sort.SliceStable(allMatches, func(i, j int) bool {
		a, b := allMatches[i], allMatches[j]
		if a.IsSemantic != b.IsSemantic {
			// Deterministic match with passed validation beats semantic prediction
			if !a.IsSemantic && a.ValidatorPassed {
				return true
			}
			if !b.IsSemantic && b.ValidatorPassed {
				return false
			}
		}
		if a.Score == b.Score {
			lenI := a.End - a.Start
			lenJ := b.End - b.Start
			return lenI > lenJ
		}
		return a.Score > b.Score
	})

	// Claim non-overlapping ranges
	claimed := make([]bool, len(text))
	var selected []detector.Match
	for _, m := range allMatches {
		overlaps := false
		for k := m.Start; k < m.End && k < len(claimed); k++ {
			if claimed[k] {
				overlaps = true
				break
			}
		}
		if !overlaps {
			for k := m.Start; k < m.End && k < len(claimed); k++ {
				claimed[k] = true
			}
			selected = append(selected, m)
		}
	}

	// 4. Sort selected matches by Start ascending for sequential token substitution
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].Start < selected[j].Start
	})

	// 5. Assign surrogate tokens (re-using token if identical or canonically equivalent value was seen in this session)
	tokenCounters := make(map[string]int)
	valueToToken := make(map[string]string)
	canonicalToToken := make(map[string]string)
	tokenMap := make(map[string]string)
	entities := make([]DetectedEntity, 0, len(selected))

	// Pre-load existing tokens for this session if any
	existing := e.vault.GetAllSessionMappings(sessionID)
	for t, orig := range existing {
		valueToToken[orig] = t
		tokenMap[t] = orig
		if inferredType := extractEntityTypeFromToken(t); inferredType != "" {
			if cKey := canonicalEntityKey(detector.EntityType(inferredType), orig); cKey != "" {
				canonicalToToken[cKey] = t
			}
		}
	}

	// Build sanitized string
	var sb strings.Builder
	sb.Grow(len(text))
	namespace := tokenNamespace(sessionID)

	cursor := 0
	for _, m := range selected {
		// Append unmasked text between matches
		if m.Start > cursor {
			sb.WriteString(text[cursor:m.Start])
		}

		label := string(m.Type)
		if m.SubType != "" {
			label = m.SubType
		}

		cKey := canonicalEntityKey(detector.EntityType(label), m.Value)
		tok, exists := valueToToken[m.Value]
		if !exists && cKey != "" {
			tok, exists = canonicalToToken[cKey]
		}
		if !exists {
			// Include a short session namespace so placeholders cannot be guessed before
			// the request is sent. The gateway uses a cryptographically random internal
			// session ID for every request.
			for {
				tokenCounters[label]++
				tok = fmt.Sprintf("[HG_%s_%s_%d]", namespace, label, tokenCounters[label])
				if _, alreadyUsed := tokenMap[tok]; !alreadyUsed {
					break
				}
			}
			valueToToken[m.Value] = tok
			if cKey != "" {
				canonicalToToken[cKey] = tok
			}
			tokenMap[tok] = m.Value
			// Save to Vault
			e.vault.Put(sessionID, tok, m.Value, label, e.ttl)
		} else {
			valueToToken[m.Value] = tok
		}

		sb.WriteString(tok)
		entities = append(entities, DetectedEntity{
			Type:         m.Type,
			SubType:      m.SubType,
			Token:        tok,
			Original:     m.Value,
			Start:        m.Start,
			End:          m.End,
			Score:        m.Score,
			ContextFound: m.ContextFound,
		})

		cursor = m.End
	}

	// Append any remaining text
	if cursor < len(text) {
		sb.WriteString(text[cursor:])
	}

	return Result{
		SanitizedText: sb.String(),
		Entities:      entities,
		TokenMap:      tokenMap,
	}
}

func tokenNamespace(sessionID string) string {
	sum := sha256.Sum256([]byte(sessionID))
	return fmt.Sprintf("%x", sum[:6])
}

// canonicalEntityKey normalizes an entity value so equivalent representations
// (e.g. "529.982.247-25" and "52998224725", or "Maria Silva" and "MARIA SILVA")
// receive the same surrogate token within a session.
func canonicalEntityKey(entityType detector.EntityType, raw string) string {
	switch entityType {
	case detector.EntityCPF, detector.EntityCNPJ, detector.EntityCreditCard, detector.EntityCNH, detector.EntityCEP,
		detector.EntitySSN, detector.EntityEIN, detector.EntityNHS, detector.EntityNIR,
		detector.EntitySUS, detector.EntityGovernmentID, detector.EntityLegal:
		var digits strings.Builder
		digits.Grow(len(raw))
		for _, r := range raw {
			if unicode.IsDigit(r) {
				digits.WriteRune(r)
			}
		}
		if digits.Len() == 0 {
			return ""
		}
		return string(entityType) + ":" + digits.String()
	case detector.EntityIBAN, detector.EntityNINO, detector.EntityDNI, detector.EntityCodiceFiscale, detector.EntityMedicalID,
		detector.EntityVehicle, detector.EntityDevice, detector.EntityCID:
		var alnum strings.Builder
		alnum.Grow(len(raw))
		for _, r := range raw {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				alnum.WriteRune(unicode.ToUpper(r))
			}
		}
		return string(entityType) + ":" + alnum.String()
	case detector.EntityEmail, detector.EntityName, detector.EntityMedication:
		normalized := strings.Join(strings.Fields(strings.ToLower(raw)), " ")
		return string(entityType) + ":" + normalized
	default:
		return string(entityType) + ":" + raw
	}
}

func extractEntityTypeFromToken(token string) string {
	// Token format: [HG_<namespace>_<TYPE>_<index>] (or legacy [PG_...])
	trimmed := strings.TrimSuffix(token, "]")
	if strings.HasPrefix(trimmed, "[HG_") {
		trimmed = strings.TrimPrefix(trimmed, "[HG_")
	} else {
		trimmed = strings.TrimPrefix(trimmed, "[PG_")
	}
	firstUnderscore := strings.IndexByte(trimmed, '_')
	lastUnderscore := strings.LastIndexByte(trimmed, '_')
	if firstUnderscore <= 0 || lastUnderscore <= firstUnderscore {
		return ""
	}
	return trimmed[firstUnderscore+1 : lastUnderscore]
}

// EgressResult contains the output of the outbound LLM response scan.
type EgressResult struct {
	SanitizedText  string           `json:"sanitized_text"`
	LeakedEntities []DetectedEntity `json:"leaked_entities"`
}

// ScanEgress inspects text emitted by an upstream LLM, masking any newly leaked
// sensitive data that was NOT present in the user's original request mappings.
func (e *Engine) ScanEgress(knownPlaintexts map[string]bool, text string) EgressResult {
	if text == "" {
		return EgressResult{SanitizedText: "", LeakedEntities: nil}
	}

	var allMatches []detector.Match

	for _, d := range e.detectors {
		matches := d.Detect(text)
		for _, m := range matches {
			if m.Score < e.ThresholdFor(m.Type) {
				continue
			}
			if e.allowlist[strings.ToLower(m.Value)] {
				continue
			}
			// Skip entities that were legitimately restored from user prompt mappings
			if knownPlaintexts != nil {
				if knownPlaintexts[m.Value] {
					continue
				}
				cKey := canonicalEntityKey(m.Type, m.Value)
				if cKey != "" && knownPlaintexts[cKey] {
					continue
				}
			}
			allMatches = append(allMatches, m)
		}
	}

	if len(allMatches) == 0 {
		return EgressResult{SanitizedText: text, LeakedEntities: nil}
	}

	sort.SliceStable(allMatches, func(i, j int) bool {
		a, b := allMatches[i], allMatches[j]
		if a.Score == b.Score {
			return (a.End - a.Start) > (b.End - b.Start)
		}
		return a.Score > b.Score
	})

	claimed := make([]bool, len(text))
	var selected []detector.Match
	for _, m := range allMatches {
		overlaps := false
		for k := m.Start; k < m.End && k < len(claimed); k++ {
			if claimed[k] {
				overlaps = true
				break
			}
		}
		if !overlaps {
			for k := m.Start; k < m.End && k < len(claimed); k++ {
				claimed[k] = true
			}
			selected = append(selected, m)
		}
	}

	sort.Slice(selected, func(i, j int) bool {
		return selected[i].Start < selected[j].Start
	})

	tokenCounters := make(map[string]int)
	var sb strings.Builder
	sb.Grow(len(text))
	cursor := 0
	var leaked []DetectedEntity

	for _, m := range selected {
		if m.Start > cursor {
			sb.WriteString(text[cursor:m.Start])
		}
		label := string(m.Type)
		if m.SubType != "" {
			label = m.SubType
		}
		tokenCounters[label]++
		tok := fmt.Sprintf("[HG_EGRESS_%s_%d]", label, tokenCounters[label])
		sb.WriteString(tok)

		leaked = append(leaked, DetectedEntity{
			Type:         m.Type,
			SubType:      m.SubType,
			Token:        tok,
			Original:     m.Value,
			Start:        m.Start,
			End:          m.End,
			Score:        m.Score,
			ContextFound: m.ContextFound,
		})
		cursor = m.End
	}

	if cursor < len(text) {
		sb.WriteString(text[cursor:])
	}

	return EgressResult{
		SanitizedText:  sb.String(),
		LeakedEntities: leaked,
	}
}
