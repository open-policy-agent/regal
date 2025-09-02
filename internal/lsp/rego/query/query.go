package query

const (
	Keywords          = "data.regal.ast.keywords"
	RuleHeadLocations = "data.regal.ast.rule_head_locations"
	Completion        = "data.regal.lsp.completion.items"
)

func AllQueries() []string {
	return []string{Keywords, RuleHeadLocations, Completion}
}
