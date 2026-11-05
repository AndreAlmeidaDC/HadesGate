package anonymizer

import (
	"strings"

	"github.com/hadesgate/hadesgate/internal/anonymizer/detector"
)

// Family represents a regulatory and architectural macro-grouping of sensitive data.
type Family string

const (
	FamilyPersonal   Family = "PERSONAL"
	FamilyIntl       Family = "INTL"
	FamilyHealth     Family = "HEALTH_PHI"
	FamilyCorporate  Family = "CORPORATE"
	FamilyLegal      Family = "LEGAL"
	FamilyVehicle    Family = "VEHICLE"
	FamilyDevice     Family = "DEVICE"
	FamilyGovernment Family = "GOVERNMENT"
	FamilySession    Family = "SESSION"
	FamilySpecial    Family = "SPECIAL"
)

// AllFamilies lists all available macro-families in the HadesGate taxonomy.
var AllFamilies = []Family{
	FamilyPersonal,
	FamilyIntl,
	FamilyHealth,
	FamilyCorporate,
	FamilyLegal,
	FamilyVehicle,
	FamilyDevice,
	FamilyGovernment,
	FamilySession,
	FamilySpecial,
}

// EntityToFamily maps each EntityType to its designated regulatory macro-family.
var EntityToFamily = map[detector.EntityType]Family{
	// PERSONAL
	detector.EntityCPF:         FamilyPersonal,
	detector.EntityCreditCard:  FamilyPersonal,
	detector.EntityPhone:       FamilyPersonal,
	detector.EntityEmail:       FamilyPersonal,
	detector.EntityName:        FamilyPersonal,
	detector.EntityCEP:         FamilyPersonal,
	detector.EntityPIX:         FamilyPersonal,
	detector.EntityRG:          FamilyPersonal,
	detector.EntityCNH:         FamilyPersonal,
	detector.EntityBankAccount: FamilyPersonal,
	detector.EntityPassport:    FamilyPersonal,

	// INTERNATIONAL IDENTIFIERS
	detector.EntityIBAN:          FamilyIntl,
	detector.EntitySSN:           FamilyIntl,
	detector.EntityNINO:          FamilyIntl,
	detector.EntityDNI:           FamilyIntl,
	detector.EntityNIR:           FamilyIntl,
	detector.EntityCodiceFiscale: FamilyIntl,

	// HEALTH & PHI (HIPAA / LGPD Art. 11)
	detector.EntityNHS:        FamilyHealth,
	detector.EntityMedicalID:  FamilyHealth,
	detector.EntitySUS:        FamilyHealth,
	detector.EntityCID:        FamilyHealth,
	detector.EntityMedication: FamilyHealth,

	// CORPORATE & BUSINESS SECRETS
	detector.EntityCNPJ:      FamilyCorporate,
	detector.EntityEIN:       FamilyCorporate,
	detector.EntityCorporate: FamilyCorporate,

	// VEHICLES
	detector.EntityVehicle: FamilyVehicle,

	// LEGAL
	detector.EntityLegal: FamilyLegal,

	// GOVERNMENT
	detector.EntityGovernmentID: FamilyGovernment,

	// INFRASTRUCTURE & DEVICES
	detector.EntityIPAddress: FamilyDevice,
	detector.EntityDevice:    FamilyDevice,

	// SECRETS & AUTHENTICATION
	detector.EntityAPIKey: FamilySession,
	detector.EntitySecret: FamilySession,

	// CUSTOM
	detector.EntityCustom: FamilyPersonal,
}

// FamilyForEntity returns the macro-family for a given entity type, defaulting to FamilyPersonal.
func FamilyForEntity(entityType detector.EntityType) Family {
	if fam, ok := EntityToFamily[entityType]; ok {
		return fam
	}
	return FamilyPersonal
}

// EntitiesInFamily returns all EntityTypes belonging to a specific family.
func EntitiesInFamily(family Family) []detector.EntityType {
	var list []detector.EntityType
	for ent, fam := range EntityToFamily {
		if fam == family {
			list = append(list, ent)
		}
	}
	return list
}

// ResolveConfidenceThreshold calculates the effective confidence score threshold.
// Precedence:
// 1. Entity-specific threshold override (e.g. "PERSON" -> 0.85)
// 2. Family-specific threshold override (e.g. "HEALTH_PHI" -> 0.60)
// 3. Global fallback threshold (default: 0.70)
func ResolveConfidenceThreshold(
	entityType detector.EntityType,
	globalMin float64,
	familyThresholds map[string]float64,
	entityThresholds map[string]float64,
) float64 {
	entKey := strings.ToUpper(string(entityType))
	if val, ok := entityThresholds[entKey]; ok && val > 0 && val <= 1.0 {
		return val
	}

	fam := FamilyForEntity(entityType)
	famKey := strings.ToUpper(string(fam))
	if val, ok := familyThresholds[famKey]; ok && val > 0 && val <= 1.0 {
		return val
	}

	if globalMin > 0 && globalMin <= 1.0 {
		return globalMin
	}
	return 0.70
}
