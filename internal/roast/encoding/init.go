package encoding

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	_ "github.com/open-policy-agent/regal/pkg/roast/intern"
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
)
