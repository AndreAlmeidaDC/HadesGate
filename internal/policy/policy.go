package policy

import "strings"

type Action string

const (
	Mask       Action = "mask"
	Block      Action = "block"
	RouteLocal Action = "route_local"
)

type Decision struct {
	Action Action
	Types  []string
}

type Engine struct {
	actions        map[string]Action
	familyResolver func(entityType string) string
}

func New(actions map[string]string) *Engine {
	compiled := make(map[string]Action, len(actions))
	for entityType, rawAction := range actions {
		action := Action(strings.ToLower(strings.TrimSpace(rawAction)))
		if action == Mask || action == Block || action == RouteLocal {
			compiled[strings.ToUpper(strings.TrimSpace(entityType))] = action
		}
	}
	return &Engine{actions: compiled}
}

// WithFamilyResolver configures an optional resolver to map entity types to macro-families for batch policy fallback.
func (e *Engine) WithFamilyResolver(fn func(entityType string) string) *Engine {
	e.familyResolver = fn
	return e
}

// Decide uses the safest priority: block, then local route, then mask.
func (e *Engine) Decide(entityTypes []string) Decision {
	decision := Decision{Action: Mask}
	seen := make(map[string]bool)
	for _, rawType := range entityTypes {
		entityType := strings.ToUpper(rawType)
		action := e.actions[entityType]
		if action == "" && e.familyResolver != nil {
			fam := strings.ToUpper(e.familyResolver(entityType))
			action = e.actions[fam]
		}
		if action == "" {
			action = Mask
		}
		if action != Mask && !seen[entityType] {
			decision.Types = append(decision.Types, entityType)
			seen[entityType] = true
		}
		if action == Block {
			decision.Action = Block
		} else if action == RouteLocal && decision.Action != Block {
			decision.Action = RouteLocal
		}
	}
	return decision
}
