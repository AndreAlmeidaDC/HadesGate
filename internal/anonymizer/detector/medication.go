package detector

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

//go:embed anvisa_catalog.gz
var embeddedAnvisaCatalogGz []byte

const (
	DefaultAnvisaETag         = "2bcaf8a-65dbd91320198"
	DefaultAnvisaLastModified = "Tue, 13 Oct 2026 19:10:26 GMT"
)

// dosageSuffixPattern matches clinical dosage and administration unit expressions following a medication name.
var dosageSuffixPattern = regexp.MustCompile(`^(?:\s+(?:[0-9]+(?:[.,][0-9]+)?\s*(?:mg|g|mcg|ml|gotas|comprimidos?|c[aá]psulas?|ampolas?)))\b`)

// ambiguousPharmaWords contains short terms or common Portuguese homonyms present in pharmaceutical registries
// that strictly require nearby clinical context (dosage, prescription, symptoms) to avoid false positives.
var ambiguousPharmaWords = map[string]bool{
	"sol": true, "ar": true, "dia": true, "bio": true, "max": true, "uno": true,
	"gel": true, "plus": true, "flex": true, "viva": true, "total": true, "acao": true,
	"forte": true, "suave": true, "sal": true, "agua": true, "zinco": true, "ferro": true,
	"calcio": true, "alcool": true, "cloreto": true, "sodio": true, "potassio": true,
	"vital": true, "life": true, "multi": true, "baby": true, "care": true, "ultra": true,
	"mega": true, "pleno": true, "sono": true, "doce": true, "puro": true, "creme": true,
	"teste": true, "test": true, "kit": true, "pasta": true, "po": true, "gas": true,
	"base": true, "vida": true, "ativo": true, "inativo": true, "similar": true, "novo": true,
	"simples": true, "composto": true, "marca": true, "produto": true, "solucao": true,
}

// MedicationCatalogStatus reports the runtime state of the pharmacological catalog.
type MedicationCatalogStatus struct {
	TermsCount   int       `json:"terms_count"`
	ActiveETag   string    `json:"active_etag"`
	LastModified string    `json:"last_modified"`
	LastChecked  time.Time `json:"last_checked"`
	Source       string    `json:"source"` // "embedded" or "live_updated"
}

// MedicationDetector detects pharmaceutical medications, active ingredients, and clinical dosages
// using a high-performance in-memory Aho-Corasick automaton compiled from the official ANVISA catalog.
type MedicationDetector struct {
	mu           sync.RWMutex
	ac           *AhoCorasick
	termsCount   int
	activeETag   string
	lastModified string
	lastChecked  time.Time
	source       string
}

// NewMedicationDetector creates and initializes a MedicationDetector with the embedded ANVISA dataset.
func NewMedicationDetector() *MedicationDetector {
	d := &MedicationDetector{
		activeETag:   DefaultAnvisaETag,
		lastModified: DefaultAnvisaLastModified,
		lastChecked:  time.Now(),
		source:       "embedded",
	}

	terms := d.loadInitialCatalog()
	d.ac = NewAhoCorasick(terms)
	d.termsCount = len(terms)

	return d
}

// Type returns EntityMedication.
func (d *MedicationDetector) Type() EntityType {
	return EntityMedication
}

// ContextWords returns pharmacological and clinical prescription context keywords.
func (d *MedicationDetector) ContextWords() []string {
	return []string{
		"medicamento", "medicamentos", "remedio", "remédio", "prescricao", "prescrição",
		"receita", "posologia", "comprimido", "comprimidos", "gotas", "capsulas", "cápsulas",
		"via oral", "injetavel", "injetável", "tratamento", "farmacia", "farmácia", "dosagem",
		"dose", "tomar", "toma", "tomando", "receitou", "prescreveu", "bula", "laboratorio",
		"laboratório", "tarja preta", "tarja vermelha", "mg", "ml", "mcg", "pomada", "xarope",
	}
}

// Status returns runtime statistics about the active pharmacological catalog.
func (d *MedicationDetector) Status() MedicationCatalogStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return MedicationCatalogStatus{
		TermsCount:   d.termsCount,
		ActiveETag:   d.activeETag,
		LastModified: d.lastModified,
		LastChecked:  d.lastChecked,
		Source:       d.source,
	}
}

// Reload updates the in-memory Aho-Corasick automaton with newly synchronized terms.
func (d *MedicationDetector) Reload(terms []string, etag, lastModified, source string) {
	if len(terms) == 0 {
		return
	}

	newAC := NewAhoCorasick(terms)

	d.mu.Lock()
	defer d.mu.Unlock()

	d.ac = newAC
	d.termsCount = len(terms)
	if etag != "" {
		d.activeETag = etag
	}
	if lastModified != "" {
		d.lastModified = lastModified
	}
	d.lastChecked = time.Now()
	if source != "" {
		d.source = source
	}
}

// Detect scans input text for medication names, active ingredients, and clinical dosages.
func (d *MedicationDetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	d.mu.RLock()
	ac := d.ac
	d.mu.RUnlock()

	if ac == nil {
		return nil
	}

	acMatches := ac.Search(text)
	if len(acMatches) == 0 {
		return nil
	}

	var matches []Match
	claimed := make(map[int]bool)

	for _, acm := range acMatches {
		start := acm.Start
		end := acm.End

		if claimed[start] || claimed[end-1] {
			continue
		}

		matchedText := text[start:end]
		normTerm := normalizeMedicationText(matchedText)

		// Check if a clinical dosage suffix follows immediately (e.g. " 500mg" or " 10 ml")
		remaining := text[end:]
		if dosageLoc := dosageSuffixPattern.FindStringIndex(remaining); dosageLoc != nil && dosageLoc[0] == 0 {
			end = end + dosageLoc[1]
			matchedText = text[start:end]
		}

		ctxText := ExtractSurroundingContext(text, start, end, 5)
		ctxFound := FindContextKeywords(ctxText, d.ContextWords())
		hasDosage := strings.ContainsAny(matchedText, "0123456789")

		isAmbiguous := len(normTerm) <= 4 || ambiguousPharmaWords[normTerm]

		// Ambiguous or short terms strictly require surrounding clinical context or attached dosage
		if isAmbiguous && len(ctxFound) == 0 && !hasDosage {
			continue
		}

		score := 0.95
		if len(ctxFound) > 0 || hasDosage {
			score = 1.0
		} else if isAmbiguous {
			score = 0.85
		}

		matches = append(matches, Match{
			Type:            EntityMedication,
			Value:           matchedText,
			Start:           start,
			End:             end,
			Score:           score,
			ContextFound:    ctxFound,
			ValidatorPassed: true,
		})

		for k := start; k < end; k++ {
			claimed[k] = true
		}
	}

	return matches
}

func (d *MedicationDetector) loadInitialCatalog() []string {
	// 1. Try loading from local persistent cache file if exists
	if data, err := os.ReadFile("data/anvisa_catalog.bin"); err == nil && len(data) > 0 {
		if terms, err := decompressGzLines(data); err == nil && len(terms) > 0 {
			d.source = "live_updated"
			return terms
		}
	}

	// 2. Fallback to embedded ANVISA catalog
	if len(embeddedAnvisaCatalogGz) > 0 {
		if terms, err := decompressGzLines(embeddedAnvisaCatalogGz); err == nil && len(terms) > 0 {
			return terms
		}
	}

	// 3. Built-in hardcoded fallback in the unlikely event of decompression error
	return []string{
		"dipirona", "amoxicilina", "paracetamol", "omeprazol", "clonazepam",
		"losartana", "metformina", "sertralina", "ibuprofeno", "fluoxetina",
		"atenolol", "enalapril", "novalgina", "rivotril", "insuman",
	}
}

func decompressGzLines(gzData []byte) ([]string, error) {
	gr, err := gzip.NewReader(bytes.NewReader(gzData))
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	decompressed, err := io.ReadAll(gr)
	if err != nil {
		return nil, err
	}

	rawLines := strings.Split(string(decompressed), "\n")
	terms := make([]string, 0, len(rawLines))
	for _, l := range rawLines {
		t := strings.TrimSpace(l)
		if t != "" {
			terms = append(terms, t)
		}
	}

	return terms, nil
}
