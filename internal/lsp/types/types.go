package types

import (
	"strconv"

	"github.com/open-policy-agent/regal/internal/lsp/types/symbols"
	"github.com/open-policy-agent/regal/internal/util"
)

type (
	FileDiagnostics struct {
		URI   string       `json:"uri"`
		Items []Diagnostic `json:"diagnostics"`
	}

	WorkspaceDidChangeWatchedFilesParams struct {
		Changes []FileEvent `json:"changes"`
	}

	FileEvent struct {
		URI  string `json:"uri"`
		Type uint   `json:"type"`
	}

	InitializationOptions struct {
		// Formatter specifies the formatter to use. Options: 'opa fmt' (default),
		// 'opa fmt --rego-v1' or 'regal fix'.
		Formatter string `json:"formatter,omitempty"`
		// EnableDebugCodelens, if set, will enable debug codelens
		// when clients request code lenses for a file.
		EnableDebugCodelens bool `json:"enableDebugCodelens,omitempty"`
		// EvalCodelensDisplayInline, if set, will show evaluation results natively
		// in the calling editor, rather than in an output file.
		EvalCodelensDisplayInline bool `json:"evalCodelensDisplayInline,omitempty"`
		// EnableExplorer, if set, will enable the regal.explorer command
		// and related functionality.
		EnableExplorer bool `json:"enableExplorer,omitempty"`
		// EnableServerTesting, if set, will enable test location notifications
		// via the regal/testLocations and test running handler.
		EnableServerTesting bool `json:"enableServerTesting,omitempty"`
	}

	ServerInfo struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	TextDocumentPositionParams struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Position     Position               `json:"position"`
	}
	DefinitionParams = TextDocumentPositionParams
	HoverParams      = TextDocumentPositionParams

	ExecuteCommandParams struct {
		Command   string `json:"command"`
		Arguments []any  `json:"arguments"`
	}

	ApplyWorkspaceEditParams struct {
		Label string        `json:"label"`
		Edit  WorkspaceEdit `json:"edit"`
	}

	ApplyWorkspaceRenameEditParams struct {
		Label string              `json:"label"`
		Edit  WorkspaceRenameEdit `json:"edit"`
	}

	ApplyWorkspaceAnyEditParams struct {
		Label string           `json:"label"`
		Edit  WorkspaceAnyEdit `json:"edit"`
	}
	WorkspaceAnyEdit struct {
		DocumentChanges []any `json:"documentChanges"`
	}

	CreateFileOptions struct {
		Overwrite      bool `json:"overwrite"`
		IgnoreIfExists bool `json:"ignoreIfExists"`
	}

	CreateFile struct {
		Options              *CreateFileOptions `json:"options,omitempty"`
		AnnotationIdentifier *string            `json:"annotationId,omitempty"`
		Kind                 string             `json:"kind"` // must always be "create"
		URI                  string             `json:"uri"`
	}

	RenameFileOptions struct {
		Overwrite      bool `json:"overwrite"`
		IgnoreIfExists bool `json:"ignoreIfExists"`
	}

	RenameFile struct {
		Options              *RenameFileOptions `json:"options,omitempty"`
		AnnotationIdentifier *string            `json:"annotationId,omitempty"`
		Kind                 string             `json:"kind"` // must always be "rename"
		OldURI               string             `json:"oldUri"`
		NewURI               string             `json:"newUri"`
	}

	DeleteFileOptions struct {
		Recursive         bool `json:"recursive"`
		IgnoreIfNotExists bool `json:"ignoreIfNotExists"`
	}

	DeleteFile struct {
		Options *DeleteFileOptions `json:"options,omitempty"`
		Kind    string             `json:"kind"` // must always be "delete"
		URI     string             `json:"uri"`
	}

	// WorkspaceRenameEdit is a WorkspaceEdit that is used for renaming files.
	// Perhaps we should use generics and a union type here instead.
	WorkspaceRenameEdit struct {
		DocumentChanges []RenameFile `json:"documentChanges"`
	}

	WorkspaceDeleteEdit struct {
		DocumentChanges []DeleteFile `json:"documentChanges"`
	}

	WorkspaceEdit struct {
		DocumentChanges []TextDocumentEdit `json:"documentChanges"`
	}

	TextDocumentEdit struct {
		// TextDocument is the document to change. Not that this could be versioned,
		// (OptionalVersionedTextDocumentIdentifier) but we currently don't use that.
		TextDocument OptionalVersionedTextDocumentIdentifier `json:"textDocument"`
		Edits        []TextEdit                              `json:"edits"`
	}

	TextEdit struct {
		NewText string `json:"newText"`
		Range   Range  `json:"range"`
	}

	TextDocumentParams struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}

	// Note(anderseknert): The LSP spec allows additional 'options' for formatting, like the number of
	// spaces to use for indentation, etc. Since we don't support any formatter other than
	// 'opa fmt' (and 'opa fmt'-compatible fixers), we don't represent that in DocumentFormattingParams.

	DocumentFormattingParams = TextDocumentParams
	DocumentSymbolParams     = TextDocumentParams
	SemanticTokensParams     = TextDocumentParams
	CodeLensParams           = TextDocumentParams

	DocumentSymbol struct {
		Detail         *string            `json:"detail,omitempty"`
		Children       *[]DocumentSymbol  `json:"children,omitempty"`
		Name           string             `json:"name"`
		Range          Range              `json:"range"`
		SelectionRange Range              `json:"selectionRange"`
		Kind           symbols.SymbolKind `json:"kind"`
	}

	WorkspaceSymbolParams struct {
		Query string `json:"query"`
	}

	WorkspaceSymbol struct {
		ContainerName *string            `json:"containerName,omitempty"`
		Name          string             `json:"name"`
		Location      Location           `json:"location"`
		Kind          symbols.SymbolKind `json:"kind"`
	}

	DidSaveTextDocumentParams struct {
		Text         *string                `json:"text,omitempty"`
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}

	TextDocumentIdentifier struct {
		URI string `json:"uri"`
	}

	OptionalVersionedTextDocumentIdentifier struct {
		// Version is optional (i.e. it can be null), but it cannot be undefined when used in some requests
		// (see workspace/applyEdit).
		Version *uint  `json:"version"`
		URI     string `json:"uri"`
	}

	DidChangeTextDocumentParams struct {
		TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
		ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
	}

	VersionedTextDocumentIdentifier struct {
		Version uint   `json:"version"`
		URI     string `json:"uri"`
	}

	TextDocumentContentChangeEvent struct {
		Range *Range `json:"range,omitempty"`
		Text  string `json:"text"`
	}

	Diagnostic struct {
		CodeDescription *CodeDescription `json:"codeDescription,omitempty"`
		Message         string           `json:"message"`
		Source          *string          `json:"source,omitempty"`
		Code            string           `json:"code"` // spec says optional integer or string
		Range           Range            `json:"range"`
		Severity        *uint            `json:"severity,omitempty"`
	}

	CodeDescription struct {
		Href string `json:"href"`
	}

	DiagnosticCode struct {
		Value  string `json:"value"`
		Target string `json:"target"`
	}

	Range struct {
		Start Position `json:"start"`
		End   Position `json:"end"`
	}

	Position struct {
		Line      uint `json:"line"`
		Character uint `json:"character"`
	}

	DidOpenTextDocumentParams struct {
		TextDocument TextDocumentItem `json:"textDocument"`
	}

	DidCloseTextDocumentParams struct {
		TextDocument TextDocumentItem `json:"textDocument"`
	}

	TextDocumentItem struct {
		LanguageID string `json:"languageId"`
		Text       string `json:"text"`
		URI        string `json:"uri"`
		Version    uint   `json:"version"`
	}

	File = TextDocumentIdentifier

	Location struct {
		URI   string `json:"uri"`
		Range Range  `json:"range"`
	}

	FilesParams struct {
		Files []File `json:"files"`
	}

	CreateFilesParams = FilesParams
	DeleteFilesParams = FilesParams

	RenameFilesParams struct {
		Files []FileRename `json:"files"`
	}

	FileRename struct {
		NewURI string `json:"newUri"`
		OldURI string `json:"oldUri"`
	}

	WorkspaceDiagnosticReport struct {
		Items []WorkspaceFullDocumentDiagnosticReport `json:"items"`
	}

	WorkspaceFullDocumentDiagnosticReport struct {
		URI     string       `json:"uri"`
		Version *uint        `json:"version"`
		Kind    string       `json:"kind"` // full, or incremental. We always use full
		Items   []Diagnostic `json:"items"`
	}

	TraceParams struct {
		Value string `json:"value"`
	}

	SemanticTokens struct {
		ResultID *string  `json:"resultId,omitempty"`
		Data     []uint32 `json:"data"`
	}

	SemanticTokensLegend struct {
		TokenTypes     []string `json:"tokenTypes"`
		TokenModifiers []string `json:"tokenModifiers"`
	}

	ExplorerCommandArgs struct {
		Target      string `json:"target"`
		Strict      bool   `json:"strict,omitempty"`
		Annotations bool   `json:"annotations,omitempty"`
		Print       bool   `json:"print,omitempty"`
		Format      bool   `json:"format,omitempty"`
	}

	ExplorerStageResult struct {
		Name   string `json:"name"`
		Output string `json:"output"`
		Error  bool   `json:"error"`
	}

	ExplorerResult struct {
		Stages []ExplorerStageResult `json:"stages"`
		Plan   string                `json:"plan,omitempty"`
	}

	iuint interface{ ~int | ~uint }
)

func (p Position) ToOffset(text string) int {
	if p.Line == 0 {
		return util.SafeUintToInt(p.Character)
	}

	if offset := util.IndexByteNth(text, '\n', p.Line); offset > -1 {
		return offset + 1 + util.SafeUintToInt(p.Character)
	}

	return len(text)
}

func (r RenameFile) AppendJSON(bs []byte) []byte {
	bs = strconv.AppendQuote(append(bs, `{"kind":"rename","oldUri":`...), r.OldURI)
	bs = strconv.AppendQuote(append(bs, `,"newUri":`...), r.NewURI)

	if r.Options != nil && (r.Options.IgnoreIfExists || r.Options.Overwrite) {
		bs = r.Options.AppendJSON(append(bs, `,"options":`...))
	}

	return append(bs, '}')
}

func (ro RenameFileOptions) AppendJSON(bs []byte) []byte {
	bs = append(bs, '{')
	if ro.IgnoreIfExists {
		if bs = append(bs, `"ignoreIfExists":true`...); ro.Overwrite {
			bs = append(bs, ',')
		}
	}

	if ro.Overwrite {
		bs = append(bs, `"overwrite":true`...)
	}

	return append(bs, '}')
}

func (d DeleteFile) AppendJSON(bs []byte) []byte {
	bs = strconv.AppendQuote(append(bs, `{"kind":"delete","uri":`...), d.URI)

	if d.Options != nil {
		bs = d.Options.AppendJSON(append(bs, `,"options":`...))
	}

	return append(bs, '}')
}

func (do DeleteFileOptions) AppendJSON(bs []byte) []byte {
	bs = append(bs, '{')
	if do.IgnoreIfNotExists {
		if bs = append(bs, `"ignoreIfNotExists":true`...); do.Recursive {
			bs = append(bs, ',')
		}
	}

	if do.Recursive {
		bs = append(bs, `"recursive":true`...)
	}

	return append(bs, '}')
}

func (c CreateFile) AppendJSON(bs []byte) []byte {
	bs = strconv.AppendQuote(append(bs, `{"kind":"create","uri":`...), c.URI)

	if c.Options != nil {
		bs = append(bs, `,"options":{`...)
		if c.Options.IgnoreIfExists {
			if bs = append(bs, `"ignoreIfExists":true`...); c.Options.Overwrite {
				bs = append(bs, ',')
			}
		}

		if c.Options.Overwrite {
			bs = append(bs, []byte(`"overwrite":true`)...)
		}

		bs = append(bs, '}')
	}

	if c.AnnotationIdentifier != nil {
		bs = strconv.AppendQuote(append(bs, `,"annotationId":`...), *c.AnnotationIdentifier)
	}

	return append(bs, '}')
}

func NewTextDocumentEdit(uri string, edits []TextEdit) TextDocumentEdit {
	return TextDocumentEdit{
		TextDocument: OptionalVersionedTextDocumentIdentifier{URI: uri},
		Edits:        edits,
	}
}

func (t TextDocumentEdit) AppendJSON(bs []byte) []byte {
	bs = strconv.AppendQuote(append(bs, `{"textDocument":{"uri":`...), t.TextDocument.URI)

	if t.TextDocument.Version != nil {
		bs = util.AppendUint(append(bs, `,"version":`...), *t.TextDocument.Version)
	} else {
		bs = append(bs, `,"version":null`...)
	}

	bs = append(bs, `},"edits":[`...)

	for i, edit := range t.Edits {
		if i > 0 {
			bs = append(bs, ',')
		}

		bs = strconv.AppendQuote(append(bs, `{"newText":`...), edit.NewText)
		bs = edit.Range.AppendJSON(append(bs, `,"range":`...))
		bs = append(bs, '}')
	}

	return append(bs, ']', '}')
}

func RangeBetween[T1, T2, T3, T4 iuint](startLine T1, startCharacter T2, endLine T3, endCharacter T4) Range {
	return Range{
		Start: Position{Line: uint(startLine), Character: uint(startCharacter)},
		End:   Position{Line: uint(endLine), Character: uint(endCharacter)},
	}
}

func (r Range) AppendJSON(bs []byte) []byte {
	bs = util.AppendUint(append(bs, `{"start":{"line":`...), r.Start.Line)
	bs = util.AppendUint(append(bs, `,"character":`...), r.Start.Character)
	bs = util.AppendUint(append(bs, `},"end":{"line":`...), r.End.Line)
	bs = util.AppendUint(append(bs, `,"character":`...), r.End.Character)

	return append(bs, '}', '}')
}
