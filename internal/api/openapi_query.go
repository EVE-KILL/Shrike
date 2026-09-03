package api

import (
	"fmt"
	"sort"

	"github.com/danielgtaylor/huma/v2"
	"github.com/eve-kill/shrike/internal/killtype"
)

// Query parameters for the read surface.
//
// registerLegacy hands the raw huma.Context to the handler and sets
// SkipValidateParams, so Huma never builds an input struct and the document
// carried no parameters at all: 329 of 341 GET operations declared nothing
// while 49 files read req.Query across 99 distinct names. An advanced killmail
// search rendered on the reference page with no way to search.
//
// The declarations live in one table rather than at each registration site for
// the same reason the descriptions do: those sites have no single shape. Some
// are huma.Operation literals, some come from helpers like entityIDOperation,
// and the SDE, entity-page, universe, and conflict surfaces are generated from
// table specs whose handlers are shared across many routes. A parameter set
// threaded through all of those would touch fifty files to say one thing.
//
// The declarations describe what the handler reads. They are documentation,
// not validation: SkipValidateParams stays on, the handlers keep reading
// req.Query, and an undeclared parameter is still accepted exactly as before.
// Nothing here can change a response.
//
// Bounds and defaults are copied from the parsing site, so "limit, default 50,
// 10-100" means boundedQueryInt(req, "limit", 50, 10, 100) rather than a
// plausible guess. Where a handler clamps rather than rejects, the maximum is
// the value you get, not an error you receive.

// entityExtension marks a parameter that names an entity, so the reference
// page offers its search picker instead of a bare number box. The path-segment
// heuristic on the page only covers path parameters; a query parameter has no
// segment to read, so the kind is stated here.
const entityExtension = "x-entity"

// applyOperationParameters attaches the declared query parameters, leaving any
// parameter already set at the registration site alone.
//
// Path parameters are declared at their registration sites and must survive:
// this appends rather than replaces, and skips a name that is already present.
func applyOperationParameters(document *huma.OpenAPI) {
	forEachOperation(document, func(operation *huma.Operation) {
		declared, ok := operationQueryParameters[operation.OperationID]
		if !ok {
			return
		}
		existing := make(map[string]bool, len(operation.Parameters))
		for _, parameter := range operation.Parameters {
			existing[parameter.In+" "+parameter.Name] = true
		}
		for _, parameter := range declared {
			if existing["query "+parameter.Name] {
				continue
			}
			operation.Parameters = append(operation.Parameters, parameter)
		}
	})
}

// checkParameterisedOperations reports IDs in the table that match no
// operation. A renamed or deleted route otherwise leaves its parameters behind
// where nothing reads them and nothing complains.
func checkParameterisedOperations(document *huma.OpenAPI) error {
	live := map[string]bool{}
	forEachOperation(document, func(operation *huma.Operation) {
		live[operation.OperationID] = true
	})
	var stale []string
	for id := range operationQueryParameters {
		if !live[id] {
			stale = append(stale, id)
		}
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		return fmt.Errorf("parameterised operations no longer exist: %v", stale)
	}
	return nil
}

// declaredQueryNames returns every query parameter name the table declares.
func declaredQueryNames() map[string]bool {
	names := map[string]bool{}
	for _, parameters := range operationQueryParameters {
		for _, parameter := range parameters {
			names[parameter.Name] = true
		}
	}
	return names
}

// --- constructors -----------------------------------------------------------

func queryParam(name, description string, schema *huma.Schema) *huma.Param {
	return &huma.Param{
		Name:        name,
		In:          "query",
		Description: description,
		Schema:      schema,
	}
}

// requiredQuery marks a parameter whose handler rejects omission.
func requiredQuery(parameter *huma.Param) *huma.Param {
	parameter.Required = true
	return parameter
}

// textQuery declares a free-text parameter.
func textQuery(name, description string) *huma.Param {
	return queryParam(name, description, &huma.Schema{Type: huma.TypeString})
}

// enumQuery declares a parameter with a closed set of values. Pass an empty
// fallback when omitting the parameter is not the same as sending a default.
func enumQuery(
	name, description, fallback string,
	values ...string,
) *huma.Param {
	schema := &huma.Schema{
		Type: huma.TypeString,
		Enum: make([]any, 0, len(values)),
	}
	for _, value := range values {
		schema.Enum = append(schema.Enum, value)
	}
	if fallback != "" {
		schema.Default = fallback
	}
	return queryParam(name, description, schema)
}

// integerEnumQuery declares a numeric selector with a closed set of values.
func integerEnumQuery(
	name, description string,
	fallback int,
	values ...int,
) *huma.Param {
	schema := &huma.Schema{
		Type:    huma.TypeInteger,
		Default: fallback,
		Enum:    make([]any, 0, len(values)),
	}
	for _, value := range values {
		schema.Enum = append(schema.Enum, value)
	}
	return queryParam(name, description, schema)
}

// intQuery declares an unbounded integer, for identifiers and cursors.
func intQuery(name, description string) *huma.Param {
	return queryParam(name, description, &huma.Schema{Type: huma.TypeInteger})
}

// numberQuery declares an unbounded JSON number.
func numberQuery(name, description string) *huma.Param {
	return queryParam(name, description, &huma.Schema{Type: huma.TypeNumber})
}

// boundedQuery declares an integer the handler clamps into a range.
func boundedQuery(
	name, description string,
	fallback, minimum, maximum int,
) *huma.Param {
	low, high := float64(minimum), float64(maximum)
	return queryParam(name, description, &huma.Schema{
		Type:    huma.TypeInteger,
		Default: fallback,
		Minimum: &low,
		Maximum: &high,
	})
}

// cappedQuery declares an integer with an upper bound but no useful floor.
func cappedQuery(
	name, description string,
	fallback, maximum int,
) *huma.Param {
	high := float64(maximum)
	return queryParam(name, description, &huma.Schema{
		Type:    huma.TypeInteger,
		Default: fallback,
		Maximum: &high,
	})
}

// cappedNumberQuery declares a JSON number with an upper bound.
func cappedNumberQuery(
	name, description string,
	fallback, maximum float64,
) *huma.Param {
	return queryParam(name, description, &huma.Schema{
		Type:    huma.TypeNumber,
		Default: fallback,
		Maximum: &maximum,
	})
}

// flooredQuery declares an integer with a default and a floor but no ceiling.
func flooredQuery(
	name, description string,
	fallback, minimum int,
) *huma.Param {
	low := float64(minimum)
	return queryParam(name, description, &huma.Schema{
		Type:    huma.TypeInteger,
		Default: fallback,
		Minimum: &low,
	})
}

// defaultedQuery declares an integer the handler defaults but never clamps.
func defaultedQuery(name, description string, fallback int) *huma.Param {
	return queryParam(name, description, &huma.Schema{
		Type:    huma.TypeInteger,
		Default: fallback,
	})
}

// rangeQuery declares an integer that is only read inside a range. Omitting it
// is not the same as sending a value, so it carries no default.
func rangeQuery(name, description string, minimum, maximum int) *huma.Param {
	low, high := float64(minimum), float64(maximum)
	return queryParam(name, description, &huma.Schema{
		Type:    huma.TypeInteger,
		Minimum: &low,
		Maximum: &high,
	})
}

// flagQuery declares a switch. Every one of these handlers compares the raw
// value to "true", so anything else leaves the filter off.
func flagQuery(name, description string) *huma.Param {
	return queryParam(name, description, &huma.Schema{Type: huma.TypeBoolean})
}

// entityQuery declares an entity identifier and names the kind, so the
// reference page can offer a name search rather than a number box.
func entityQuery(name, description, kind string) *huma.Param {
	parameter := intQuery(name, description)
	parameter.Extensions = map[string]any{entityExtension: kind}
	return parameter
}

// jsonQuery declares a parameter whose value is a JSON document. The advanced
// searches carry their whole filter tree this way.
func jsonQuery(name, description string) *huma.Param {
	return queryParam(name, description, &huma.Schema{
		Type:   huma.TypeString,
		Format: "json",
	})
}

// timestampQuery declares an ISO-8601 instant.
func timestampQuery(name, description string) *huma.Param {
	return queryParam(name, description, &huma.Schema{
		Type:   huma.TypeString,
		Format: "date-time",
	})
}

// --- parameters shared across many operations --------------------------------

const cursorAdvice = "Pass the previous response's pagination cursor to " +
	"fetch the next page."

func afterQuery() *huma.Param {
	return intQuery("after", "Ascending cursor. "+cursorAdvice)
}

func beforeQuery() *huma.Param {
	return intQuery(
		"before",
		"Descending cursor, walking newest to oldest. "+cursorAdvice+
			" Mutually exclusive with `after`, which it overrides.",
	)
}

func cursorQuery() *huma.Param {
	return intQuery("cursor", "Identifier cursor. "+cursorAdvice)
}

func limitQuery(fallback, minimum, maximum int) *huma.Param {
	return boundedQuery(
		"limit", "Maximum results to return.", fallback, minimum, maximum,
	)
}

func pageQuery(fallback, minimum, maximum int) *huma.Param {
	return boundedQuery(
		"page", "Page number, counted from 1.", fallback, minimum, maximum,
	)
}

// openPageQuery declares a page number the handler floors at 1 but never caps.
func openPageQuery() *huma.Param {
	return flooredQuery("page", "Page number, counted from 1.", 1, 1)
}

const offsetPageAdvice = "Page number for offset paging. " +
	"Leave at 0 to page by cursor."

// offsetPageQuery declares the page parameter on routes that also take a
// cursor, where page 0 means "use the cursor instead".
func offsetPageQuery(maximum int) *huma.Param {
	return boundedQuery("page", offsetPageAdvice, 0, 0, maximum)
}

// openOffsetPageQuery is offsetPageQuery on a handler that applies no ceiling.
func openOffsetPageQuery() *huma.Param {
	return flooredQuery("page", offsetPageAdvice, 0, 0)
}

func daysQuery(fallback, minimum, maximum int) *huma.Param {
	return boundedQuery(
		"days", "Size of the trailing window, in days.",
		fallback, minimum, maximum,
	)
}

// killTypeQuery declares the killmail category filter shared by the kill
// lists and the top lists. The values come from the predicate table the
// handlers query, so the document cannot list a category the API rejects.
func killTypeQuery() *huma.Param {
	names := make([]string, 0, len(killtype.Predicates()))
	for name := range killtype.Predicates() {
		names = append(names, name)
	}
	sort.Strings(names)
	return enumQuery(
		"type",
		"Killmail category: space, ship class, tech level, or value band.",
		"latest",
		names...,
	)
}

func victimFactionsQuery() *huma.Param {
	return textQuery(
		"victimFactions",
		"Comma-separated faction IDs to restrict the victim to, "+
			"for example `500001,500002`.",
	)
}
