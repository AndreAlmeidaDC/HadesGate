package policy

import "strings"

type Action string

const (
	Mask       Action = "mask"
	Block      Action = "block"
	RouteLocal Action = "route_local"
	Disabled   Action = "disabled"
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
		if action == Mask || action == Block || action == RouteLocal || action == Disabled {
			compiled[strings.ToUpper(strings.TrimSpace(entityType))] = action
		}
	}
	return &Engine{actions: compiled}
}

// ActionFor returns the effective action for an entity, including macro-family fallback.
func (e *Engine) ActionFor(rawType string) Action {
	entityType := strings.ToUpper(strings.TrimSpace(rawType))
	action := e.actions[entityType]
	if action == "" && e.familyResolver != nil {
		family := strings.ToUpper(strings.TrimSpace(e.familyResolver(entityType)))
		action = e.actions[family]
	}
	if action == "" {
		return Mask
	}
	return action
}

// IsDisabled reports whether an entity must bypass DLP detection and tokenization.
func (e *Engine) IsDisabled(entityType string) bool {
	return e.ActionFor(entityType) == Disabled
}

// HasDisabledActions reports whether this policy can allow any plaintext category through.
func (e *Engine) HasDisabledActions() bool {
	for _, action := range e.actions {
		if action == Disabled {
			return true
		}
	}
	return false
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
		action := e.ActionFor(entityType)
		if action == Disabled {
			continue
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
