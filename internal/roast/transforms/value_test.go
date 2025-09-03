package transforms

import (
	"embed"
	"testing"

	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/internal/testutil"
	"github.com/open-policy-agent/regal/pkg/roast/encoding"
)

//go:embed testdata
var testData embed.FS

func TestRoastAndOPAInterfaceToValueSameOutput(t *testing.T) {
	t.Parallel()

	inputMap := inputMap(t)
	roastValue := testutil.Must(AnyToValue(inputMap))(t)
	opaValue := testutil.Must(ast.InterfaceToValue(inputMap))(t)

	if roastValue.Compare(opaValue) != 0 {
		t.Fatal("values are not equal")
	}
}

// BenchmarkInterfaceToValue-10    	 741	   1615548 ns/op	 1376979 B/op	   24189 allocs/op
// ...
func BenchmarkInterfaceToValue(b *testing.B) {
	inputMap := inputMap(b)

	for b.Loop() {
		if _, err := AnyToValue(inputMap); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkOPAInterfaceToValue-10    	616	   1942695 ns/op	 1566569 B/op	   45901 allocs/op
// BenchmarkOPAInterfaceToValue-10    	626	   1838247 ns/op	 1566848 B/op	   36037 allocs/op OPA 1.0
// ...
func BenchmarkOPAInterfaceToValue(b *testing.B) {
	inputMap := inputMap(b)

	for b.Loop() {
		if _, err := ast.InterfaceToValue(inputMap); err != nil {
			b.Fatal(err)
		}
	}
}

func inputMap(tb testing.TB) map[string]any {
	tb.Helper()

	content := string(testutil.Must(testData.ReadFile("testdata/ast.rego"))(tb))
	module := testutil.Must(ast.ParseModuleWithOpts("ast.rego", content, ast.ParserOptions{ProcessAnnotation: true}))(tb)

	return testutil.Must(encoding.JSONRoundTripTo[map[string]any](module))(tb)
}
