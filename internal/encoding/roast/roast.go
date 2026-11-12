package roast

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/open-policy-agent/opa/v1/ast"
	outil "github.com/open-policy-agent/opa/v1/util"

	"github.com/open-policy-agent/regal/internal/encoding/util/write"
	"github.com/open-policy-agent/regal/internal/funsafe"
	"github.com/open-policy-agent/regal/internal/util"
	"github.com/open-policy-agent/regal/pkg/roast/rast"
)

var (
	Marshalers = json.JoinMarshalers(
		json.MarshalToFunc(AnnotationsMarshalToFn),
		json.MarshalToFunc(ModuleMarshalToFn),
		json.MarshalToFunc(PackageMarshalToFn),
		json.MarshalToFunc(ImportMarshalToFn),
		json.MarshalToFunc(RuleMarshalToFn),
		json.MarshalToFunc(HeadMarshalToFn),
		json.MarshalToFunc(BodyMarshalToFn),
		json.MarshalToFunc(ExprMarshalToFn),
		json.MarshalToFunc(RefMarshalToFn),
		json.MarshalToFunc(CallMarshalToFn),
		json.MarshalToFunc(TermMarshalToFn),
		json.MarshalToFunc(ValueMarshalToFn),
		json.MarshalToFunc(SomeDeclMarshalToFn),
		json.MarshalToFunc(EveryMarshalToFn),
		json.MarshalToFunc(WithMarshalToFn),
		json.MarshalToFunc(NotMarshalToFn),
		json.MarshalToFunc(LogicalAndMarshalToFn),
		json.MarshalToFunc(LogicalOrMarshalToFn),
		json.MarshalToFunc(CommentMarshalToFn),
		json.MarshalToFunc(LocationMarshalToFn),
		json.MarshalToFunc(ArrayMarshalToFn),
		json.MarshalToFunc(ArrayComprehensionMarshalToFn),
		json.MarshalToFunc(ObjectComprehensionMarshalToFn),
		json.MarshalToFunc(SetComprehensionMarshalToFn),
		json.MarshalToFunc(TemplateStringMarshalToFn),
		json.MarshalToFunc(SetMarshalToFn),
		json.MarshalToFunc(ObjectMarshalToFn),
	)
	Options = json.JoinOptions(
		json.WithMarshalers(Marshalers),
		json.WithUnmarshalers(json.UnmarshalFromFunc(NumberUnmarshalFromFunc)),
		// NOTE(anders): This is obviously not a good practice when processing
		// arbitrary JSON in either direction. However, the risk of duplicate
		// names is close to nil as we process modules parsed by OPA, or by our
		// own tools. Running with this check enabled decreases ns/op by about
		// 18% in marshaling performance, which is too significant to ignore.
		jsontext.AllowDuplicateNames(true),
	)
	pkgDataTerm = jsontext.Value(`{"type":"var","value":"data"}`)
)

func ModuleMarshalToFn(enc *jsontext.Encoder, mod *ast.Module) (err error) {
	enc.WriteToken(jsontext.BeginObject)
	write.String(enc, "package")

	_ = writePackage(enc, mod.Package, util.Filter(mod.Annotations, notDocumentOrRuleScope))

	write.ArrayFieldFn(enc, "imports", mod.Imports, ImportMarshalToFn)
	write.ArrayFieldFn(enc, "rules", mod.Rules, RuleMarshalToFn)
	write.ArrayFieldFn(enc, "comments", mod.Comments, CommentMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}

func PackageMarshalToFn(enc *jsontext.Encoder, p *ast.Package) (err error) {
	return writePackage(enc, p, nil)
}

func ImportMarshalToFn(enc *jsontext.Encoder, imp *ast.Import) (err error) {
	write.ObjectStart(enc, imp.Location)
	write.FieldFn(enc, "path", imp.Path, TermMarshalToFn)
	write.StringField(enc, "alias", imp.Alias)

	return enc.WriteToken(jsontext.EndObject)
}

func AnnotationsMarshalToFn(enc *jsontext.Encoder, a *ast.Annotations) error {
	write.ObjectStart(enc, a.Location)
	write.StringField(enc, "scope", a.Scope)
	write.StringField(enc, "title", a.Title)
	write.StringField(enc, "description", a.Description)
	write.WhenTrue(enc, a.Entrypoint, "entrypoint")
	write.ArrayOfStringField(enc, "organizations", a.Organizations)
	write.ArrayFieldTo(enc, "related_resources", a.RelatedResources)
	write.ArrayField(enc, "authors", a.Authors)
	write.ArrayField(enc, "schemas", a.Schemas)
	write.MapField(enc, "custom", a.Custom)

	return enc.WriteToken(jsontext.EndObject)
}

func CommentMarshalToFn(enc *jsontext.Encoder, c *ast.Comment) error {
	// Marshal only location string — text is retrieved dynamically via regal.file.lines
	return LocationMarshalToFn(enc, c.Location)
}

func LocationMarshalToFn(enc *jsontext.Encoder, location *ast.Location) (err error) {
	endRow, endCol := location.End()

	buf := append(enc.AvailableBuffer(), '"')
	buf = append(outil.AppendInt(buf, location.Row), ':')
	buf = append(outil.AppendInt(buf, location.Col), ':')
	buf = append(outil.AppendInt(buf, endRow), ':')
	buf = append(outil.AppendInt(buf, endCol), '"')

	return enc.WriteValue(buf)
}

func ElseMarshalToFn(enc *jsontext.Encoder, rule *ast.Rule) error {
	write.ObjectStart(enc, rule.Location)
	write.FieldPtrFn(enc, "head", rule.Head, ElseHeadMarshalToFn)

	if !rast.IsBodyGenerated(rule) {
		write.FieldFn(enc, "body", rule.Body, BodyMarshalToFn)
	}

	write.FieldPtrFn(enc, "else", rule.Else, ElseMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}

func RuleMarshalToFn(enc *jsontext.Encoder, rule *ast.Rule) error {
	write.ObjectStart(enc, rule.Location)
	write.ArrayFieldFn(enc, "annotations", rule.Annotations, AnnotationsMarshalToFn)
	write.WhenTrue(enc, rule.Default, "default")
	write.FieldPtrFn(enc, "head", rule.Head, HeadMarshalToFn)

	if !rast.IsBodyGenerated(rule) {
		write.FieldFn(enc, "body", rule.Body, BodyMarshalToFn)
	}

	write.FieldPtrFn(enc, "else", rule.Else, ElseMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}

func ElseHeadMarshalToFn(enc *jsontext.Encoder, head *ast.Head) error {
	write.ObjectStart(enc, head.Location)
	write.WhenTrue(enc, head.Assign, "assign")

	if head.Value != nil {
		// Strip location from generated `true` values, as they don't have one
		if head.Value.Location != nil &&
			head.Location != nil &&
			head.Value.Location.Row == head.Location.Row &&
			head.Value.Location.Col == head.Location.Col {
			head.Value.Location = nil
		}

		write.FieldFn(enc, "value", head.Value, TermMarshalToFn)
	}

	return enc.WriteToken(jsontext.EndObject)
}

func HeadMarshalToFn(enc *jsontext.Encoder, head *ast.Head) error {
	write.ObjectStart(enc, head.Location)
	write.ArrayFieldFn(enc, "ref", head.Reference, TermMarshalToFn)
	write.ArrayFieldFn(enc, "args", head.Args, TermMarshalToFn)
	write.WhenTrue(enc, head.Assign, "assign")
	write.FieldPtrFn(enc, "key", head.Key, TermMarshalToFn)

	if head.Value != nil {
		// Strip location from generated `true` values, as they don't have one
		if head.Value.Location != nil &&
			head.Location != nil &&
			head.Value.Location.Row == head.Location.Row &&
			head.Value.Location.Col == head.Location.Col {
			head.Value.Location = nil
		}

		write.FieldFn(enc, "value", head.Value, TermMarshalToFn)
	}

	return enc.WriteToken(jsontext.EndObject)
}

func BodyMarshalToFn(enc *jsontext.Encoder, body ast.Body) error {
	return write.ArrayToFn(enc, body, ExprMarshalToFn)
}

func ExprMarshalToFn(enc *jsontext.Encoder, expr *ast.Expr) error {
	return writeExpr(enc, expr, false)
}

func WithMarshalToFn(enc *jsontext.Encoder, with *ast.With) error {
	write.ObjectStart(enc, with.Location)
	write.FieldFn(enc, "target", with.Target, TermMarshalToFn)
	write.FieldFn(enc, "value", with.Value, TermMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}

func LogicalAndMarshalToFn(enc *jsontext.Encoder, and *ast.LogicalAnd) error {
	write.ObjectStart(enc, and.Location)
	write.StringField(enc, "type", "and")
	write.WhenTrue(enc, and.ExplicitLhs, "explicit_lhs")
	write.WhenTrue(enc, and.ExplicitRhs, "explicit_rhs")
	write.ArrayFieldFn(enc, "lhs", and.Lhs, ExprMarshalToFn)
	write.ArrayFieldFn(enc, "rhs", and.Rhs, ExprMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}

func LogicalOrMarshalToFn(enc *jsontext.Encoder, or *ast.LogicalOr) error {
	write.ObjectStart(enc, or.Location)
	write.StringField(enc, "type", "or")
	write.WhenTrue(enc, or.ExplicitLhs, "explicit_lhs")
	write.WhenTrue(enc, or.ExplicitRhs, "explicit_rhs")
	write.ArrayFieldFn(enc, "lhs", or.Lhs, ExprMarshalToFn)
	write.ArrayFieldFn(enc, "rhs", or.Rhs, ExprMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}

func TermMarshalToFn(enc *jsontext.Encoder, term *ast.Term) error {
	write.ObjectStart(enc, term.Location)
	write.StringField(enc, "type", ast.ValueName(term.Value))
	write.FieldFn(enc, "value", term.Value, ValueMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}

func ValueMarshalToFn(enc *jsontext.Encoder, value ast.Value) (err error) {
	switch t := value.(type) {
	case ast.Null:
		return enc.WriteToken(jsontext.Null)
	case ast.Number:
		return enc.WriteValue(outil.StringToByteSlice(t))
	case ast.Boolean:
		if t {
			return enc.WriteToken(jsontext.True)
		}

		return enc.WriteToken(jsontext.False)
	case ast.String:
		return write.String(enc, t)
	case ast.Var:
		return write.String(enc, t)
	case ast.Object:
		return ObjectMarshalToFn(enc, t)
	case *ast.Array:
		return ArrayMarshalToFn(enc, t)
	case ast.Set:
		return SetMarshalToFn(enc, t)
	case *ast.Not:
		return NotMarshalToFn(enc, t)
	case ast.Ref:
		return RefMarshalToFn(enc, t)
	case ast.Call:
		return CallMarshalToFn(enc, t)
	case *ast.TemplateString:
		return TemplateStringMarshalToFn(enc, t)
	case *ast.ArrayComprehension:
		return ArrayComprehensionMarshalToFn(enc, t)
	case *ast.SetComprehension:
		return SetComprehensionMarshalToFn(enc, t)
	case *ast.ObjectComprehension:
		return ObjectComprehensionMarshalToFn(enc, t)
	default:
		return json.MarshalEncode(enc, t)
	}
}

func NotMarshalToFn(enc *jsontext.Encoder, not *ast.Not) error {
	write.ObjectStart(enc, not.Location)
	write.StringField(enc, "type", "not")
	write.WhenTrue(enc, not.ExplicitBody, "explicit_body")
	write.ArrayFieldToFn(enc, "body", not.Body, ExprMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}

func ArrayMarshalToFn(enc *jsontext.Encoder, arr *ast.Array) error {
	enc.WriteToken(jsontext.BeginArray)

	for i := range arr.Len() {
		if err := TermMarshalToFn(enc, arr.Elem(i)); err != nil {
			return err
		}
	}

	return enc.WriteToken(jsontext.EndArray)
}

func ObjectMarshalToFn(enc *jsontext.Encoder, obj ast.Object) error {
	enc.WriteToken(jsontext.BeginArray)

	elems := funsafe.ObjectElems(obj)
	for i := range elems {
		enc.WriteToken(jsontext.BeginArray)

		if err := TermMarshalToFn(enc, elems[i].Key()); err != nil {
			return err
		}

		if err := TermMarshalToFn(enc, elems[i].Value()); err != nil {
			return err
		}

		enc.WriteToken(jsontext.EndArray)
	}

	return enc.WriteToken(jsontext.EndArray)
}

func SetMarshalToFn(enc *jsontext.Encoder, s ast.Set) (err error) {
	return write.ArrayToFn(enc, s.Slice(), TermMarshalToFn)
}

func RefMarshalToFn(enc *jsontext.Encoder, r ast.Ref) (err error) {
	return write.ArrayToFn(enc, r, TermMarshalToFn)
}

func CallMarshalToFn(enc *jsontext.Encoder, call ast.Call) (err error) {
	return write.ArrayToFn(enc, call, TermMarshalToFn)
}

func NumberUnmarshalFromFunc(dec *jsontext.Decoder, num *ast.Number) error {
	val, err := dec.ReadValue()
	if err == nil {
		*num = ast.Number(string(val))
	}

	return err
}

func SomeDeclMarshalToFn(enc *jsontext.Encoder, some *ast.SomeDecl) error {
	write.ObjectStart(enc, some.Location)
	write.ArrayFieldFn(enc, "symbols", some.Symbols, TermMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}

func EveryMarshalToFn(enc *jsontext.Encoder, every *ast.Every) error {
	write.ObjectStart(enc, every.Location)
	write.FieldPtrFn(enc, "key", every.Key, TermMarshalToFn)
	write.FieldFn(enc, "value", every.Value, TermMarshalToFn)
	write.FieldFn(enc, "domain", every.Domain, TermMarshalToFn)
	write.FieldFn(enc, "body", every.Body, BodyMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}

func TemplateStringMarshalToFn(enc *jsontext.Encoder, ts *ast.TemplateString) (err error) {
	enc.WriteToken(jsontext.BeginObject)
	write.WhenTrue(enc, ts.MultiLine, "multi_line")
	write.Tokens(enc, jsontext.String("parts"), jsontext.BeginArray)

	for _, part := range ts.Parts {
		if expr, ok := part.(*ast.Expr); ok {
			err = writeExpr(enc, expr, true)
		} else {
			err = json.MarshalEncode(enc, part)
		}

		if err != nil {
			return err
		}
	}

	return write.Tokens(enc, jsontext.EndArray, jsontext.EndObject)
}

func ArrayComprehensionMarshalToFn(enc *jsontext.Encoder, ac *ast.ArrayComprehension) error {
	enc.WriteToken(jsontext.BeginObject)
	write.FieldFn(enc, "term", ac.Term, TermMarshalToFn)
	write.FieldFn(enc, "body", ac.Body, BodyMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}

func SetComprehensionMarshalToFn(enc *jsontext.Encoder, sc *ast.SetComprehension) error {
	enc.WriteToken(jsontext.BeginObject)
	write.FieldFn(enc, "term", sc.Term, TermMarshalToFn)
	write.FieldFn(enc, "body", sc.Body, BodyMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}

func ObjectComprehensionMarshalToFn(enc *jsontext.Encoder, oc *ast.ObjectComprehension) error {
	enc.WriteToken(jsontext.BeginObject)
	write.FieldFn(enc, "key", oc.Key, TermMarshalToFn)
	write.FieldFn(enc, "value", oc.Value, TermMarshalToFn)
	write.FieldFn(enc, "body", oc.Body, BodyMarshalToFn)

	return enc.WriteToken(jsontext.EndObject)
}

func writePackage(enc *jsontext.Encoder, p *ast.Package, annotations []*ast.Annotations) (err error) {
	write.ObjectStart(enc, p.Location)

	if path := p.Path; len(path) > 0 {
		write.String(enc, "path")
		enc.WriteToken(jsontext.BeginArray)

		if value, ok := path[0].Value.(ast.Var); ok && value == "data" {
			enc.WriteValue(pkgDataTerm)

			path = path[1:]
		}

		write.Foreach(enc, path, TermMarshalToFn)
		enc.WriteToken(jsontext.EndArray)
	}

	if len(annotations) > 0 {
		write.ArrayFieldFn(enc, "annotations", annotations, AnnotationsMarshalToFn)
	}

	return enc.WriteToken(jsontext.EndObject)
}

func writeExpr(enc *jsontext.Encoder, expr *ast.Expr, interpolated bool) (err error) {
	write.ObjectStart(enc, expr.Location)
	write.WhenTrue(enc, expr.Negated, "negated")
	write.WhenTrue(enc, expr.Generated, "generated")
	write.WhenTrue(enc, interpolated, "interpolated")

	if len(expr.With) > 0 {
		write.ArrayFieldFn(enc, "with", expr.With, WithMarshalToFn)
	}

	if expr.Terms != nil {
		write.String(enc, "terms")

		switch t := expr.Terms.(type) {
		case *ast.Term:
			err = TermMarshalToFn(enc, t)
		case []*ast.Term:
			err = write.ArrayToFn(enc, t, TermMarshalToFn)
		case *ast.SomeDecl:
			err = SomeDeclMarshalToFn(enc, t)
		case *ast.Every:
			err = EveryMarshalToFn(enc, t)
		case *ast.Not:
			err = NotMarshalToFn(enc, t)
		case *ast.LogicalAnd:
			err = LogicalAndMarshalToFn(enc, t)
		case *ast.LogicalOr:
			err = LogicalOrMarshalToFn(enc, t)
		}

		if err != nil {
			return err
		}
	}

	return enc.WriteToken(jsontext.EndObject)
}

func notDocumentOrRuleScope(a *ast.Annotations) bool {
	return a.Scope != "document" && a.Scope != "rule"
}
