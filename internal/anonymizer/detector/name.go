package detector

import (
	"regexp"
	"strings"
	"unicode"
)

// contextualNameRegex finds names preceded by greetings, honorifics, role labels, action verbs, and prepositions.
// Case-insensitivity (?i:...) is scoped strictly to the trigger and connectors so lowercase prose words after a name are never swallowed.
var contextualNameRegex = regexp.MustCompile(
	`\b((?i:` +
		`bom\s+dia|boa\s+tarde|boa\s+noite|nome\s+completo|nome(?:\s+[ée]|\s*:)?|chama-se|assinado\s+por|` +
		`cobran[çc]a\s+para|cobrar(?:\s+de)?|fatura\s+(?:de|para)|faturar\s+para|pagamento\s+(?:de|para)|` +
		`transferir\s+para|transfer[êe]ncia\s+para|enviar\s+para|pix\s+para|boleto\s+(?:de|para)|` +
		`cadastro\s+de|dados\s+de|conta\s+de|cpf\s+de|rg\s+de|cnh\s+de|contrato\s+(?:de|para)|` +
		`ol[áa]|oi|oie|prezad[oa]s?|car[oa]s?|estimad[oa]s?|senhor(?:a|es|as)?|sr\.?|sra\.?|srta\.?|` +
		`dr\.?|dra\.?|doutor(?:a|es|as)?|prof\.?|profa\.?|professor(?:a|es|as)?|eng\.?|adv\.?|` +
		`cliente|paciente|autor(?:a|es|as)?|usu[aá]ri[oa]s?|atendente|colaborador(?:a|es|as)?|` +
		`funcion[aá]ri[oa]s?|servidor(?:a|es|as)?|titular|portador(?:a|es|as)?|benefici[aá]ri[oa]s?|` +
		`favorecid[oa]s?|pagador(?:a|es|as)?|devedor(?:a|es|as)?|recebedor(?:a|es|as)?|destinat[aá]ri[oa]s?|` +
		`remetente|comprador(?:a|es|as)?|vendedor(?:a|es|as)?|locat[aá]ri[oa]s?|locador(?:a|es|as)?|` +
		`fiador(?:a|es|as)?|representante|s[óo]ci[oa]s?|motorista|condutor(?:a|es|as)?|segurad[oa]s?|` +
		`dependente|respons[aá]vel|candidat[oa]s?|alun[oa]s?|estudante|para|de|do|da|com|por|ao` +
		`))\s*[,:]?\s+` +
		`((?:[A-ZÀ-Ú][a-zà-ú]+|[A-ZÀ-Ú]{2,})(?:\s+(?i:da|de|do|das|dos|e)\s+(?:[A-ZÀ-Ú][a-zà-ú]+|[A-ZÀ-Ú]{2,})|\s+(?:[A-ZÀ-Ú][a-zà-ú]+|[A-ZÀ-Ú]{2,})){0,5})\b`,
)

// standaloneNameRegex captures 2-to-6 word proper name candidates anywhere in text (e.g. at start of sentence
// before a document apposition like "Rodolfo Bandeira de CPF: 09897684492" or "Rodolfo Bandeira, CPF ...").
var standaloneNameRegex = regexp.MustCompile(
	`\b((?:[A-ZÀ-Ú][a-zà-ú]+|[A-ZÀ-Ú]{2,})(?:\s+(?i:da|de|do|das|dos|e)\s+(?:[A-ZÀ-Ú][a-zà-ú]+|[A-ZÀ-Ú]{2,})|\s+(?:[A-ZÀ-Ú][a-zà-ú]+|[A-ZÀ-Ú]{2,})){1,5})\b`,
)

// rightDocAppositionRegex detects personal document/attribute labels immediately following a person's name
// (with or without comma/preposition), such as "de CPF: ...", ", RG ...", "CPF 098...", "(CNH ...)".
var rightDocAppositionRegex = regexp.MustCompile(
	`^\s*(?:[,:\-\(\/]\s*)?(?:(?i:de|do|da|com|no|na|sob|portador(?:a)?\s+d[eo]|inscrit[oa]\s+n[oa]|titular\s+d[eo])\s+)?` +
		`(?i:cpf|rg|cnh|pis|pasep|nis|nit|ctps|passaporte|matr[ií]cula|nascid[oa]|data\s+de\s+nascimento|e-?mail|telefone|celular|whatsapp|chave\s+pix)\b`,
)

var weakNamePrepositions = map[string]bool{
	"para": true, "de": true, "do": true, "da": true, "com": true, "por": true, "ao": true,
}

// locationPrepositions are weak evidence for a place rather than a person when
// no explicit person trigger or document apposition is present.
var locationPrepositions = map[string]bool{
	"em": true, "entre": true,
}

var connectorParticles = map[string]bool{
	"da": true, "de": true, "do": true, "das": true, "dos": true, "e": true,
}

// leadingTriggerWords are role/action words that may be capitalized at the start of a sentence
// immediately before a name (e.g. "Cliente Rodolfo Bandeira", "Motorista João da Silva").
var leadingTriggerWords = map[string]bool{
	"ola": true, "oi": true, "oie": true, "prezado": true, "prezada": true, "prezados": true, "prezadas": true,
	"caro": true, "cara": true, "estimado": true, "estimada": true, "senhor": true, "senhora": true,
	"sr": true, "sra": true, "srta": true, "dr": true, "dra": true, "doutor": true, "doutora": true,
	"prof": true, "profa": true, "professor": true, "professora": true, "cliente": true, "paciente": true,
	"autor": true, "autora": true, "usuario": true, "usuaria": true, "atendente": true,
	"colaborador": true, "colaboradora": true, "funcionario": true, "funcionaria": true,
	"servidor": true, "servidora": true, "titular": true, "portador": true, "portadora": true,
	"beneficiario": true, "beneficiaria": true, "favorecido": true, "favorecida": true,
	"pagador": true, "pagadora": true, "devedor": true, "devedora": true, "recebedor": true, "recebedora": true,
	"destinatario": true, "destinataria": true, "remetente": true, "comprador": true, "compradora": true,
	"vendedor": true, "vendedora": true, "locatario": true, "locataria": true, "locador": true, "locadora": true,
	"fiador": true, "fiadora": true, "representante": true, "socio": true, "socia": true,
	"motorista": true, "condutor": true, "condutora": true, "segurado": true, "segurada": true,
	"dependente": true, "responsavel": true, "candidato": true, "candidata": true, "aluno": true, "aluna": true,
	"estudante": true, "nome": true, "cobrar": true, "faturar": true,
}

// blockingPrefixWords are words that, when immediately preceding a capitalized token, indicate a place,
// street, institution, or document rather than a person's name (e.g. "São Paulo", "Hospital Albert", "Rua João").
var blockingPrefixWords = map[string]bool{
	"sao": true, "santa": true, "santo": true, "porto": true, "belo": true, "rio": true, "minas": true,
	"mato": true, "espirito": true, "distrito": true, "banco": true, "receita": true, "policia": true,
	"nota": true, "chave": true, "boleto": true, "hospital": true, "clinica": true, "universidade": true,
	"faculdade": true, "colegio": true, "escola": true, "instituto": true, "fundacao": true, "prefeitura": true,
	"tribunal": true, "ministerio": true, "secretaria": true, "rua": true, "avenida": true, "av": true,
	"praca": true, "alameda": true, "travessa": true, "rodovia": true, "estrada": true, "shopping": true,
	"condominio": true, "edificio": true, "aeroporto": true, "estacao": true, "vila": true, "parque": true,
	"bairro": true, "cidade": true, "municipio": true, "estado": true, "projeto": true, "operacao": true,
	"versao": true, "lote": true, "pedido": true,
}

// nonNameWords filters out documents, acronyms, places, institutions, payment terms, and common prose words.
// All keys are normalized (lowercase, unaccented).
var nonNameWords = map[string]bool{
	// Document & Identifier Labels
	"cpf": true, "cnpj": true, "rg": true, "cnh": true, "pis": true, "pasep": true,
	"nis": true, "nit": true, "ctps": true, "cep": true, "iban": true, "pix": true,
	"ssp": true, "uf": true, "passaporte": true, "documento": true, "identidade": true,
	"carteira": true, "registro": true, "matricula": true, "protocolo": true, "titulo": true,
	"eleitor": true, "certidao": true, "dn": true, "id": true, "uuid": true,
	// Financial, Banking & Billing
	"banco": true, "central": true, "itau": true, "bradesco": true, "santander": true,
	"caixa": true, "nubank": true, "inter": true, "btg": true, "sicoob": true, "sicredi": true,
	"safra": true, "picpay": true, "agencia": true, "conta": true, "corrente": true,
	"poupanca": true, "cartao": true, "credito": true, "debito": true, "fatura": true,
	"boleto": true, "bancario": true, "bancaria": true, "cobranca": true, "pagamento": true,
	"transferencia": true, "ted": true, "doc": true, "valor": true, "total": true, "saldo": true,
	"extrato": true, "recibo": true, "comprovante": true, "nota": true, "fiscal": true,
	"nfe": true, "nfse": true, "danfe": true, "chave": true, "imposto": true, "taxa": true,
	"juros": true, "multa": true, "desconto": true, "parcela": true, "mensalidade": true,
	// Corporate & Institutions
	"ltda": true, "sa": true, "s/a": true, "mei": true, "epp": true, "eireli": true,
	"holding": true, "grupo": true, "empresa": true, "companhia": true, "cia": true,
	"corporacao": true, "associacao": true, "fundacao": true, "instituto": true,
	"ministerio": true, "secretaria": true, "prefeitura": true, "governo": true,
	"uniao": true, "receita": true, "federal": true, "estadual": true, "municipal": true,
	"policia": true, "tribunal": true, "justica": true, "juizado": true, "vara": true,
	"comarca": true, "cartorio": true, "oficio": true, "universidade": true, "faculdade": true,
	"escola": true, "colegio": true, "hospital": true, "clinica": true, "laboratorio": true,
	"farmacia": true, "loja": true, "mercado": true, "supermercado": true, "shopping": true,
	"condominio": true, "edificio": true, "acme": true,
	// Geography & Addresses
	"sao": true, "rio": true, "minas": true, "gerais": true, "santa": true, "catarina": true,
	"parana": true, "bahia": true, "goias": true, "pernambuco": true, "ceara": true,
	"maranhao": true, "amazonas": true, "espirito": true, "paraiba": true, "mato": true,
	"grosso": true, "alagoas": true, "piaui": true, "distrito": true, "sergipe": true,
	"rondonia": true, "tocantins": true, "acre": true, "amapa": true, "roraima": true,
	"porto": true, "alegre": true, "belo": true, "horizonte": true, "curitiba": true,
	"brasilia": true, "recife": true, "salvador": true, "fortaleza": true, "manaus": true,
	"belem": true, "goiania": true, "campinas": true, "guarulhos": true, "niteroi": true,
	"florianopolis": true, "maceio": true, "natal": true, "teresina": true, "brasil": true,
	"sul": true, "norte": true, "leste": true, "oeste": true, "endereco": true, "rua": true,
	"avenida": true, "av": true, "alameda": true, "praca": true, "travessa": true,
	"rodovia": true, "estrada": true, "numero": true, "complemento": true, "apto": true,
	"apartamento": true, "bloco": true, "andar": true, "bairro": true, "cidade": true,
	"estado": true, "pais": true,
	// Contact, Web & Tech
	"email": true, "e-mail": true, "mail": true, "telefone": true, "celular": true,
	"fone": true, "tel": true, "whatsapp": true, "contato": true, "site": true, "url": true,
	"http": true, "https": true, "api": true, "key": true, "token": true, "secret": true,
	"senha": true, "password": true, "login": true, "username": true, "ip": true,
	"dns": true, "ssh": true, "ssl": true, "tls": true, "jwt": true, "oauth": true,
	"json": true, "xml": true, "html": true, "css": true, "sql": true, "db": true,
	"database": true, "server": true, "cloud": true, "docker": true, "kubernetes": true,
	"linux": true, "windows": true, "macos": true, "android": true, "ios": true,
	"github": true, "gitlab": true, "google": true, "openai": true, "anthropic": true,
	"claude": true, "gemini": true, "chatgpt": true, "gpt": true, "codex": true,
	"hadesgate": true, "privagate": true, "pasteguard": true, "presidio": true,
	"go": true, "golang": true, "python": true, "node": true, "react": true,
	// Months, Days & Discourse / Prose Words
	"janeiro": true, "fevereiro": true, "marco": true, "abril": true, "maio": true,
	"junho": true, "julho": true, "agosto": true, "setembro": true, "outubro": true,
	"novembro": true, "dezembro": true, "segunda": true, "terca": true, "quarta": true,
	"quinta": true, "sexta": true, "sabado": true, "domingo": true, "feira": true,
	"dia": true, "tarde": true, "noite": true, "bom": true, "boa": true,
	"tudo": true, "bem": true, "todos": true, "todas": true, "amigos": true, "amigas": true,
	"pessoal": true, "galera": true, "time": true, "equipe": true, "suporte": true,
	"atendimento": true, "sistema": true, "este": true, "esta": true, "esse": true,
	"essa": true, "aqui": true, "hoje": true, "ontem": true, "amanha": true, "agora": true,
	"voce": true, "voces": true, "projeto": true, "contrato": true, "dados": true,
	"servico": true, "servicos": true, "produto": true, "produtos": true, "pedido": true,
	"versao": true, "codigo": true, "lote": true, "relatorio": true, "arquivo": true,
	"emitir": true, "gerar": true, "criar": true, "enviar": true, "transferir": true,
	"pagar": true, "solicitar": true, "confirmar": true, "atualizar": true, "via": true,
	"seja": true, "sejam": true, "obrigado": true, "obrigada": true, "favor": true,
	"conforme": true, "acordo": true, "termo": true, "termos": true,
	"segunda-feira": true, "terca-feira": true, "quarta-feira": true, "quinta-feira": true, "sexta-feira": true,
}

// surnameSuffixes captures characteristic Portuguese, Romance, Germanic, and Slavic surname morphology.
var surnameSuffixes = []string{
	"eira", "eiro", "quinho", "quinha", "inho", "inha", "oso", "osa",
	"atti", "otti", "etti", "ucci", "elli", "ini", "ato",
	"mann", "berg", "stein", "ski", "sky", "wicz", "canti", "cante", "buquerque",
}

// NameDetector identifies person names deterministically using an embedded ~330k names catalog,
// Aho-Corasick candidate search, boundary & contextual validators, compound name grouping, and evidence scoring.
type NameDetector struct {
	ac *NameAhoCorasick
}

// NewNameDetector creates a new instance of NameDetector with the embedded Aho-Corasick automaton.
func NewNameDetector() *NameDetector {
	return &NameDetector{
		ac: GetNameAhoCorasick(),
	}
}

// Type returns EntityName ("PERSON").
func (d *NameDetector) Type() EntityType {
	return EntityName
}

// ContextWords returns keywords that provide contextual evidence for person names.
func (d *NameDetector) ContextWords() []string {
	return []string{
		"ola", "olá", "oi", "bom dia", "boa tarde", "boa noite", "prezado", "prezada", "caro", "cara",
		"nome", "sr", "sra", "dr", "dra", "cliente", "paciente", "usuario", "atendente", "pessoa",
		"colaborador", "funcionario", "titular", "portador", "beneficiario", "favorecido", "motorista",
		"cpf", "rg", "cnh",
	}
}

type wordSpan struct {
	word  string
	norm  string
	start int
	end   int
}

// Detect scans text for person names using:
// 1. Aho-Corasick candidate search over the 330,000-entry catalog.
// 2. Boundary and contextual validation (blocking non-name words, places, cities, and prose).
// 3. PERSON entity resolver to group adjacent given names, connectors, and surnames into compound names.
// 4. Fallback contextual evaluation for unlisted/foreign proper names in strong contextual patterns.
func (d *NameDetector) Detect(text string) []Match {
	if text == "" {
		return nil
	}

	var matches []Match

	// Pass 1: Aho-Corasick Candidate Search + Compound Name Clustering
	acMatches := d.detectFromAhoCorasick(text)
	for _, m := range acMatches {
		matches = mergeNameMatch(matches, m)
	}

	// Pass 2: Contextual Grammar Candidates (covers unlisted/foreign names backed by strong triggers)
	ctxMatches := d.detectFromContextualGrammar(text)
	for _, m := range ctxMatches {
		matches = mergeNameMatch(matches, m)
	}

	return matches
}

type validToken struct {
	NameCandidateToken
	isGiven     bool
	isSurname   bool
	prevWord    string
	leftTrigger string
}

func (d *NameDetector) detectFromAhoCorasick(text string) []Match {
	if d.ac == nil {
		return nil
	}

	tokens := d.ac.SearchCandidateTokens(text)
	if len(tokens) == 0 {
		return nil
	}

	var validTokens []validToken
	for _, tok := range tokens {
		// Tokens must be Title Case ("Carlos") or ALL-CAPS ("CARLOS"). All-lowercase tokens are ordinary prose.
		if !tok.IsTitleCase && !tok.IsAllCaps {
			continue
		}
		if nonNameWords[tok.Norm] || leadingTriggerWords[tok.Norm] {
			continue
		}

		prevWord := extractPreviousWordNorm(text, tok.Start)
		if prevWord != "" && blockingPrefixWords[prevWord] {
			continue
		}

		var trig string
		if prevWord != "" && leadingTriggerWords[prevWord] {
			trig = prevWord
		}

		validTokens = append(validTokens, validToken{
			NameCandidateToken: tok,
			isGiven:            (tok.Flags & NameTypeGiven) != 0,
			isSurname:          (tok.Flags & NameTypeSurname) != 0,
			prevWord:           prevWord,
			leftTrigger:        trig,
		})
	}

	if len(validTokens) == 0 {
		return nil
	}

	// Cluster adjacent candidate tokens and connector particles into compound name candidates
	type compoundGroup struct {
		start       int
		end         int
		tokens      []validToken
		hasParticle bool
		leftTrig    string
	}

	var groups []compoundGroup
	i := 0
	for i < len(validTokens) {
		grp := compoundGroup{
			start:    validTokens[i].Start,
			end:      validTokens[i].End,
			tokens:   []validToken{validTokens[i]},
			leftTrig: validTokens[i].leftTrigger,
		}

		j := i + 1
		for j < len(validTokens) {
			currEnd := grp.end
			nextStart := validTokens[j].Start

			if nextStart <= currEnd {
				// Overlapping token, advance
				j++
				continue
			}

			gap := text[currEnd:nextStart]
			trimmedGap := strings.TrimSpace(gap)

			// Adjacent tokens separated by single whitespace
			if trimmedGap == "" {
				grp.end = validTokens[j].End
				grp.tokens = append(grp.tokens, validTokens[j])
				j++
				continue
			}

			// Tokens connected by a connector particle ("da", "de", "do", "das", "dos", "e")
			normGap := normalizeToken(trimmedGap)
			if connectorParticles[normGap] {
				grp.end = validTokens[j].End
				grp.tokens = append(grp.tokens, validTokens[j])
				grp.hasParticle = true
				j++
				continue
			}

			// Disconnected
			break
		}

		groups = append(groups, grp)
		i = j
	}

	var matches []Match
	for _, grp := range groups {
		m, ok := d.evaluateCompoundGroup(text, grp.start, grp.end, grp.tokens, grp.leftTrig)
		if ok {
			matches = append(matches, m)
		}
	}

	return matches
}

func (d *NameDetector) evaluateCompoundGroup(text string, start, end int, tokens []validToken, leftTrig string) (Match, bool) {
	if len(tokens) == 0 {
		return Match{}, false
	}

	// Check if preceded immediately by a blocking prefix (e.g. "São Paulo", "Rua Silva")
	prevWord := extractPreviousWordNorm(text, start)
	if prevWord != "" {
		if blockingPrefixWords[prevWord] {
			return Match{}, false
		}
		if leftTrig == "" && leadingTriggerWords[prevWord] {
			leftTrig = prevWord
		}
	}

	value := text[start:end]
	tailSlice := text[end:]
	if len(tailSlice) > 64 {
		tailSlice = tailSlice[:64]
	}
	hasRightDocCtx := rightDocAppositionRegex.MatchString(tailSlice)
	hasStrongLeftTrig := leftTrig != "" && !weakNamePrepositions[leftTrig]
	if locationPrepositions[prevWord] && !hasStrongLeftTrig && !hasRightDocCtx {
		return Match{}, false
	}

	// Count substantive tokens and evaluate given/surname distribution and frequency evidence
	substantiveCount := len(tokens)
	hasGiven := false
	hasSurname := false
	var maxFreq uint32

	for _, tok := range tokens {
		if tok.isGiven {
			hasGiven = true
		}
		if tok.isSurname || hasSurnameMorphology(tok.Norm) {
			hasSurname = true
		}
		if tok.FreqGiven > maxFreq {
			maxFreq = tok.FreqGiven
		}
		if tok.FreqSurname > maxFreq {
			maxFreq = tok.FreqSurname
		}
	}

	score := 0.0

	if substantiveCount == 1 {
		// Single token name candidate
		// Requires strong contextual backing (greeting/role trigger or document apposition)
		if hasStrongLeftTrig {
			if maxFreq >= 10000 {
				score = 0.90
			} else {
				score = 0.85
			}
		} else if hasRightDocCtx {
			score = 0.85
		} else {
			// Isolated single token without context: score below 0.70 threshold to avoid false positives
			score = 0.60
		}
	} else {
		// Compound full name (2 or more substantive tokens)
		switch {
		case (hasStrongLeftTrig || hasRightDocCtx) && (hasGiven && hasSurname):
			score = 0.95
		case hasStrongLeftTrig || hasRightDocCtx:
			score = 0.95
		case hasGiven && hasSurname:
			if maxFreq >= 10000 {
				score = 0.90
			} else {
				score = 0.88
			}
		case hasGiven:
			score = 0.85
		default:
			score = 0.75
		}
	}

	if score < 0.70 {
		return Match{}, false
	}

	ctxText := ExtractSurroundingContext(text, start, end, 5)
	combinedCtx := strings.ToLower(leftTrig + " " + value + " " + ctxText)
	ctxFound := FindContextKeywords(combinedCtx, d.ContextWords())

	return Match{
		Type:            EntityName,
		Value:           value,
		Start:           start,
		End:             end,
		Score:           score,
		ContextFound:    ctxFound,
		ValidatorPassed: true,
	}, true
}

func (d *NameDetector) detectFromContextualGrammar(text string) []Match {
	type rawCandidate struct {
		trigger   string
		nameStart int
		nameEnd   int
		fromCtx   bool
	}

	var candidates []rawCandidate

	// Pass 1: Left-context triggered candidates
	for _, sub := range contextualNameRegex.FindAllStringSubmatchIndex(text, -1) {
		if len(sub) >= 6 && sub[4] != -1 && sub[5] != -1 {
			trig := normalizeToken(strings.TrimSpace(text[sub[2]:sub[3]]))
			candidates = append(candidates, rawCandidate{
				trigger:   trig,
				nameStart: sub[4],
				nameEnd:   sub[5],
				fromCtx:   true,
			})
		}
	}

	// Pass 2: Standalone / right-context candidates (e.g. sentence-initial names before "de CPF:")
	for _, sub := range standaloneNameRegex.FindAllStringSubmatchIndex(text, -1) {
		if len(sub) >= 4 && sub[2] != -1 && sub[3] != -1 {
			candidates = append(candidates, rawCandidate{
				trigger:   "",
				nameStart: sub[2],
				nameEnd:   sub[3],
				fromCtx:   false,
			})
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	var matches []Match
	for _, cand := range candidates {
		m, ok := d.evaluateCandidate(text, cand.trigger, cand.nameStart, cand.nameEnd, cand.fromCtx)
		if !ok {
			continue
		}
		matches = mergeNameMatch(matches, m)
	}

	return matches
}

func (d *NameDetector) evaluateCandidate(text, trigger string, rawStart, rawEnd int, fromCtx bool) (Match, bool) {
	spans := tokenizeSpan(text, rawStart, rawEnd)
	if len(spans) == 0 {
		return Match{}, false
	}

	// Peel leading role/greeting words if captured inside a standalone span (e.g. "Cliente Rodolfo Bandeira")
	for len(spans) > 1 && leadingTriggerWords[spans[0].norm] {
		trigger = spans[0].norm
		fromCtx = true
		spans = spans[1:]
	}

	// Reject if preceded immediately by a place, street, or institution prefix (e.g. "São Paulo", "Rua Rodolfo")
	previousWord := extractPreviousWordNorm(text, spans[0].start)
	if previousWord != "" {
		if blockingPrefixWords[previousWord] {
			return Match{}, false
		}
		if !fromCtx && leadingTriggerWords[previousWord] {
			trigger = previousWord
			fromCtx = true
		}
	}

	// Validate first token: if it is a stopword, connector, or invalid shape, reject the span
	first := spans[0]
	firstAllCaps := isAllUpperToken(first.word)
	if connectorParticles[first.norm] || nonNameWords[first.norm] || leadingTriggerWords[first.norm] || !isValidNameTokenShape(first.word, firstAllCaps) {
		return Match{}, false
	}

	// Scan subsequent tokens left-to-right and truncate before any document acronym, stopword, or case switch
	validLen := 1
	for i := 1; i < len(spans); i++ {
		sp := spans[i]
		if connectorParticles[sp.norm] {
			validLen = i + 1
			continue
		}
		// Stop before known non-name words, document types (CPF, RG, CNH, PIX...), places, or institutions
		if nonNameWords[sp.norm] || leadingTriggerWords[sp.norm] {
			break
		}
		// In a Title-Case name ("Rodolfo Bandeira"), an ALL-CAPS token ("CPF", "RG", "SSP", "SP", "LTDA")
		// is an acronym/document/state boundary, never a surname
		if !firstAllCaps && isAllUpperToken(sp.word) {
			break
		}
		if firstAllCaps && !isAllUpperToken(sp.word) {
			break
		}
		if !isValidNameTokenShape(sp.word, firstAllCaps) {
			break
		}
		validLen = i + 1
	}
	spans = spans[:validLen]

	// Strip any trailing connector particles ("de", "da", "do", "das", "dos", "e") left dangling after truncation
	for len(spans) > 0 && connectorParticles[spans[len(spans)-1].norm] {
		spans = spans[:len(spans)-1]
	}
	if len(spans) == 0 {
		return Match{}, false
	}

	trimmedStart := spans[0].start
	trimmedEnd := spans[len(spans)-1].end
	trimmedValue := text[trimmedStart:trimmedEnd]

	// Count substantive name tokens (excluding connectors) and evaluate catalog & morphological signals
	nameTokenCount := 0
	hasKnownGiven := false
	hasKnownSurname := false
	hasMorphSurname := false
	var maxFreq uint32

	for i, sp := range spans {
		if connectorParticles[sp.norm] {
			continue
		}
		nameTokenCount++
		if d.ac != nil {
			if entry, ok := d.ac.Lookup(sp.norm); ok {
				if (entry.Flags & NameTypeGiven) != 0 {
					hasKnownGiven = true
				}
				if (entry.Flags & NameTypeSurname) != 0 {
					hasKnownSurname = true
				}
				if entry.FreqGiven > maxFreq {
					maxFreq = entry.FreqGiven
				}
				if entry.FreqSurname > maxFreq {
					maxFreq = entry.FreqSurname
				}
			} else if i >= 1 && hasSurnameMorphology(sp.norm) {
				hasMorphSurname = true
			}
		} else if i >= 1 && hasSurnameMorphology(sp.norm) {
			hasMorphSurname = true
		}
	}

	if nameTokenCount == 0 {
		return Match{}, false
	}

	hasKnownLexicon := hasKnownGiven || hasKnownSurname
	tailSlice := text[trimmedEnd:]
	if len(tailSlice) > 64 {
		tailSlice = tailSlice[:64]
	}
	hasRightDocCtx := rightDocAppositionRegex.MatchString(tailSlice)
	isWeakTrig := weakNamePrepositions[trigger]
	hasStrongLeftTrig := fromCtx && trigger != "" && !isWeakTrig
	if locationPrepositions[previousWord] && !hasStrongLeftTrig && !hasRightDocCtx {
		return Match{}, false
	}

	// Confidence scoring matrix combining bi-directional context, lexicon, and morphology
	score := 0.0
	if nameTokenCount == 1 {
		// Single-word names require a strong left trigger (e.g. "Olá Rodolfo", "Sr. Silva")
		if hasStrongLeftTrig {
			if hasKnownLexicon {
				score = 0.90
			} else {
				score = 0.80
			}
		} else if fromCtx && isWeakTrig && hasRightDocCtx && hasKnownGiven {
			score = 0.85
		}
	} else {
		// Multi-token name candidate (2+ substantive tokens)
		if firstAllCaps && !hasKnownLexicon && !(hasStrongLeftTrig && hasRightDocCtx) {
			return Match{}, false
		}

		switch {
		case (hasStrongLeftTrig || hasRightDocCtx) && (hasKnownLexicon || hasMorphSurname):
			score = 0.95
		case hasStrongLeftTrig || hasRightDocCtx:
			// Unlisted/international proper name backed by strong left role/action trigger or right document apposition
			score = 0.90
		case fromCtx && isWeakTrig && (hasKnownLexicon || hasMorphSurname):
			if hasKnownGiven && (hasKnownSurname || hasMorphSurname) {
				score = 0.95
			} else {
				score = 0.88
			}
		case !fromCtx && hasKnownGiven && (hasKnownSurname || hasMorphSurname):
			// Standalone full name with known given name + known/morphological surname
			score = 0.88
		}
	}

	if score < 0.70 {
		return Match{}, false
	}

	ctxText := ExtractSurroundingContext(text, trimmedStart, trimmedEnd, 5)
	combinedCtx := strings.ToLower(trigger + " " + trimmedValue + " " + ctxText)
	ctxFound := FindContextKeywords(combinedCtx, d.ContextWords())

	return Match{
		Type:            EntityName,
		Value:           trimmedValue,
		Start:           trimmedStart,
		End:             trimmedEnd,
		Score:           score,
		ContextFound:    ctxFound,
		ValidatorPassed: len(ctxFound) > 0 || hasKnownLexicon || hasMorphSurname || hasRightDocCtx,
	}, true
}

func mergeNameMatch(existing []Match, candidate Match) []Match {
	for i, m := range existing {
		// Check if byte ranges overlap
		if candidate.Start < m.End && candidate.End > m.Start {
			candLen := candidate.End - candidate.Start
			existLen := m.End - m.Start
			if candidate.Score > m.Score || (candidate.Score == m.Score && candLen > existLen) {
				existing[i] = candidate
			}
			return existing
		}
	}
	return append(existing, candidate)
}

func tokenizeSpan(text string, start, end int) []wordSpan {
	var spans []wordSpan
	inWord := false
	wordStart := start
	for i := start; i < end; {
		r, size := decodeRuneAt(text, i)
		if !unicode.IsSpace(r) {
			if !inWord {
				inWord = true
				wordStart = i
			}
		} else if inWord {
			w := text[wordStart:i]
			spans = append(spans, wordSpan{
				word:  w,
				norm:  normalizeToken(w),
				start: wordStart,
				end:   i,
			})
			inWord = false
		}
		i += size
	}
	if inWord {
		w := text[wordStart:end]
		spans = append(spans, wordSpan{
			word:  w,
			norm:  normalizeToken(w),
			start: wordStart,
			end:   end,
		})
	}
	return spans
}

func extractPreviousWordNorm(text string, pos int) string {
	i := pos
	for i > 0 {
		r, size := decodeLastRuneBefore(text, i)
		if unicode.IsSpace(r) {
			i -= size
			continue
		}
		break
	}
	wordEnd := i
	for i > 0 {
		r, size := decodeLastRuneBefore(text, i)
		if unicode.IsLetter(r) || r == '-' {
			i -= size
			continue
		}
		break
	}
	if i >= wordEnd {
		return ""
	}
	return normalizeToken(text[i:wordEnd])
}

func isValidNameTokenShape(word string, allowAllCaps bool) bool {
	runes := []rune(word)
	if len(runes) < 2 {
		return false
	}
	if !unicode.IsUpper(runes[0]) {
		return false
	}

	hasVowel := false
	hasLower := false
	for idx, r := range runes {
		if r == '-' || r == '\'' {
			continue
		}
		if !unicode.IsLetter(r) {
			return false
		}
		if idx > 0 && unicode.IsLower(r) {
			hasLower = true
		}
		switch unicode.ToLower(r) {
		case 'a', 'e', 'i', 'o', 'u', 'y',
			'á', 'à', 'â', 'ã', 'ä', 'é', 'è', 'ê', 'ë',
			'í', 'ì', 'î', 'ï', 'ó', 'ò', 'ô', 'õ', 'ö',
			'ú', 'ù', 'û', 'ü':
			hasVowel = true
		}
	}
	if !hasVowel {
		return false
	}
	if !allowAllCaps && !hasLower {
		return false
	}
	// Reject Roman numerals commonly attached to titles/laws (II, III, IV, VI, VII, VIII, IX, XI, XII)
	if allowAllCaps && isRomanNumeral(word) {
		return false
	}
	return true
}

func isAllUpperToken(word string) bool {
	hasLetter := false
	for _, r := range word {
		if unicode.IsLetter(r) {
			hasLetter = true
			if !unicode.IsUpper(r) {
				return false
			}
		}
	}
	return hasLetter
}

func isRomanNumeral(word string) bool {
	for _, r := range word {
		switch r {
		case 'I', 'V', 'X', 'L', 'C', 'D', 'M':
		default:
			return false
		}
	}
	return len(word) >= 2
}

func hasSurnameMorphology(norm string) bool {
	if len(norm) < 5 || nonNameWords[norm] {
		return false
	}
	for _, suf := range surnameSuffixes {
		if strings.HasSuffix(norm, suf) {
			return true
		}
	}
	// Patronymics ending in -es / -ez (e.g. Rodrigues, Fernandes, Gonçalves, Antunes, Vasques, Hernandez)
	if len(norm) >= 6 && (strings.HasSuffix(norm, "es") || strings.HasSuffix(norm, "ez")) {
		return true
	}
	return false
}

// normalizeToken lowercases a string and strips Latin diacritics for fast accent-insensitive lookup.
func normalizeToken(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		r = unicode.ToLower(r)
		switch r {
		case 'á', 'à', 'â', 'ã', 'ä':
			r = 'a'
		case 'é', 'è', 'ê', 'ë':
			r = 'e'
		case 'í', 'ì', 'î', 'ï':
			r = 'i'
		case 'ó', 'ò', 'ô', 'õ', 'ö':
			r = 'o'
		case 'ú', 'ù', 'û', 'ü':
			r = 'u'
		case 'ç':
			r = 'c'
		case 'ñ':
			r = 'n'
		case '.':
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func decodeRuneAt(s string, i int) (rune, int) {
	for _, r := range s[i:] {
		return r, len(string(r))
	}
	return 0, 0
}

func decodeLastRuneBefore(s string, i int) (rune, int) {
	sub := s[:i]
	var last rune
	for _, r := range sub {
		last = r
	}
	return last, len(string(last))
}
