package types

import (
	"encoding/json"

	"github.com/open-policy-agent/regal/internal/lsp/types/completion"
	"github.com/open-policy-agent/regal/internal/lsp/types/symbols"
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
		Formatter *string `json:"formatter,omitempty"`
		// EnableDebugCodelens, if set, will enable debug codelens
		// when clients request code lenses for a file.
		EnableDebugCodelens *bool `json:"enableDebugCodelens,omitempty"`
		// EvalCodelensDisplayInline, if set, will show evaluation results natively
		// in the calling editor, rather than in an output file.
		EvalCodelensDisplayInline *bool `json:"evalCodelensDisplayInline,omitempty"`
	}

	InitializeParams struct {
		InitializationOptions *InitializationOptions `json:"initializationOptions,omitempty"`
		ClientInfo            ClientInfo             `json:"clientInfo"`
		Locale                string                 `json:"locale"`
		RootPath              string                 `json:"rootPath"`
		RootURI               string                 `json:"rootUri"`
		Trace                 string                 `json:"trace"`
		WorkspaceFolders      *[]WorkspaceFolder     `json:"workspaceFolders"`
		Capabilities          *json.RawMessage       `json:"capabilities,omitempty"`
		ProcessID             int                    `json:"processId"`
	}

	WorkspaceFolder struct {
		URI  string `json:"uri"`
		Name string `json:"name"`
	}

	ClientInfo struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	ShowMessageParams struct {
		Message string `json:"message"`
		Type    uint   `json:"type"`
	}

	StaleRequestSupportClientCapabilities struct {
		RetryOnContentModified []string `json:"retryOnContentModified"`
		Cancel                 bool     `json:"cancel"`
	}

	InitializeResult struct {
		Capabilities ServerCapabilities `json:"capabilities"`
	}

	ServerCapabilities struct {
		CodeLensProvider           ResolveProviderOption   `json:"codeLensProvider"`
		Workspace                  WorkspaceOptions        `json:"workspace"`
		DiagnosticProvider         DiagnosticOptions       `json:"diagnosticProvider"`
		CodeActionProvider         CodeActionOptions       `json:"codeActionProvider"`
		ExecuteCommandProvider     ExecuteCommandOptions   `json:"executeCommandProvider"`
		TextDocumentSyncOptions    TextDocumentSyncOptions `json:"textDocumentSync"`
		CompletionProvider         CompletionOptions       `json:"completionProvider"`
		InlayHintProvider          ResolveProviderOption   `json:"inlayHintProvider"`
		DocumentLinkProvider       ResolveProviderOption   `json:"documentLinkProvider"`
		SignatureHelpProvider      SignatureHelpOptions    `json:"signatureHelpProvider"`
		DocumentHighlightProvider  bool                    `json:"documentHighlightProvider"`
		HoverProvider              bool                    `json:"hoverProvider"`
		DocumentFormattingProvider bool                    `json:"documentFormattingProvider"`
		FoldingRangeProvider       bool                    `json:"foldingRangeProvider"`
		DocumentSymbolProvider     bool                    `json:"documentSymbolProvider"`
		WorkspaceSymbolProvider    bool                    `json:"workspaceSymbolProvider"`
		DefinitionProvider         bool                    `json:"definitionProvider"`
	}

	TextDocumentPositionParams struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Position     Position               `json:"position"`
	}
	DefinitionParams        = TextDocumentPositionParams
	TextDocumentHoverParams = TextDocumentPositionParams

	CompletionOptions struct {
		CompletionItem    CompletionItemOptions `json:"completionItem"`
		ResolveProvider   bool                  `json:"resolveProvider"`
		TriggerCharacters []string              `json:"triggerCharacters,omitempty"`
	}

	CompletionItemOptions struct {
		LabelDetailsSupport bool `json:"labelDetailsSupport"`
	}

	CompletionParams struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Position     Position               `json:"position"`
		Context      *CompletionContext     `json:"context,omitempty"`
	}

	CompletionContext struct {
		TriggerCharacter string                 `json:"triggerCharacter"`
		TriggerKind      completion.TriggerKind `json:"triggerKind"`
	}

	CompletionList struct {
		Items        []CompletionItem `json:"items"`
		IsIncomplete bool             `json:"isIncomplete"`
	}

	CompletionItem struct {
		LabelDetails    *CompletionItemLabelDetails `json:"labelDetails,omitempty"`
		Documentation   *MarkupContent              `json:"documentation,omitempty"`
		TextEdit        *TextEdit                   `json:"textEdit,omitempty"`
		InserTextFormat *uint                       `json:"insertTextFormat,omitempty"`
		SortText        *string                     `json:"sortText,omitempty"`
		// Regal is used to store regal-specific metadata about the completion item.
		// This is not part of the LSP spec, but used in the manager to post process
		// items before returning them to the client.
		Regal  *CompletionItemRegalMetadata `json:"_regal,omitempty"`
		Label  string                       `json:"label"`
		Detail string                       `json:"detail"`
		// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#completionItemKind
		Kind      completion.ItemKind `json:"kind"`
		Preselect bool                `json:"preselect"`
	}

	CompletionItemRegalMetadata struct {
		Provider string `json:"provider"`
	}

	CompletionItemLabelDetails struct {
		Description string `json:"description"`
		Detail      string `json:"detail"`
	}

	WorkspaceFoldersServerCapabilities struct {
		Supported bool `json:"supported"`
	}

	WorkspaceOptions struct {
		FileOperations   FileOperationsServerCapabilities   `json:"fileOperations"`
		WorkspaceFolders WorkspaceFoldersServerCapabilities `json:"workspaceFolders"`
	}

	CodeActionOptions struct {
		CodeActionKinds []string `json:"codeActionKinds"`
	}

	CodeActionParams struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Context      CodeActionContext      `json:"context"`
		Range        Range                  `json:"range"`
	}

	CodeActionContext struct {
		Diagnostics []Diagnostic `json:"diagnostics"`
		Only        []string     `json:"only,omitempty"`
		TriggerKind *uint8       `json:"triggerKind,omitempty"`
	}

	CodeAction struct {
		Command     Command      `json:"command"`
		IsPreferred *bool        `json:"isPreferred,omitempty"`
		Title       string       `json:"title"`
		Kind        string       `json:"kind"`
		Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
	}

	CodeLens struct {
		Command *Command `json:"command,omitempty"`
		Data    *any     `json:"data,omitempty"`
		Range   Range    `json:"range"`
	}

	Command struct {
		Arguments *[]any `json:"arguments,omitempty"`
		Title     string `json:"title"`
		Tooltip   string `json:"tooltip"`
		Command   string `json:"command"`
	}

	DocumentHighlightParams = TextDocumentPositionParams

	DocumentLink struct {
		Range   Range  `json:"range"`
		Target  string `json:"target,omitempty"`
		Tooltip string `json:"tooltip,omitempty"`
	}

	DocumentHighlight struct {
		Range Range `json:"range"`
		Kind  uint  `json:"kind"`
	}

	ExecuteCommandOptions struct {
		Commands []string `json:"commands"`
	}

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

	DocumentFormattingParams struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Options      FormattingOptions      `json:"options"`
	}

	TextDocumentParams struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}
	DocumentSymbolParams = TextDocumentParams
	FoldingRangeParams   = TextDocumentParams
	DocumentLinkParams   = TextDocumentParams
	CodeLensParams       = TextDocumentParams

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

	FoldingRange struct {
		StartCharacter *uint  `json:"startCharacter,omitempty"`
		EndCharacter   *uint  `json:"endCharacter,omitempty"`
		Kind           string `json:"kind"`
		StartLine      uint   `json:"startLine"`
		EndLine        uint   `json:"endLine"`
	}

	FormattingOptions struct {
		TabSize                uint `json:"tabSize"`
		InsertSpaces           bool `json:"insertSpaces"`
		TrimTrailingWhitespace bool `json:"trimTrailingWhitespace"`
		InsertFinalNewline     bool `json:"insertFinalNewline"`
		TrimFinalNewlines      bool `json:"trimFinalNewlines"`
	}

	FileOperationsServerCapabilities struct {
		DidCreate FileOperationRegistrationOptions `json:"didCreate"`
		DidRename FileOperationRegistrationOptions `json:"didRename"`
		DidDelete FileOperationRegistrationOptions `json:"didDelete"`
	}

	FileOperationRegistrationOptions struct {
		Filters []FileOperationFilter `json:"filters"`
	}

	FileOperationFilter struct {
		Scheme  string               `json:"scheme"`
		Pattern FileOperationPattern `json:"pattern"`
	}
	FileOperationPattern struct {
		Glob string `json:"glob"`
	}

	DiagnosticOptions struct {
		Identifier            string `json:"identifier"`
		InterFileDependencies bool   `json:"interFileDependencies"`
		WorkspaceDiagnostics  bool   `json:"workspaceDiagnostics"`
	}

	// ResolveProviderOption is used by a number of providers in place of a boolean value.
	// Note that at this point in time, we don't see a need for using resolver providers,
	// so this option is always set to false.
	ResolveProviderOption struct {
		ResolveProvider bool `json:"resolveProvider"`
	}

	InlayHint struct {
		Tooltip      MarkupContent `json:"tooltip"`
		Label        string        `json:"label"`
		Position     Position      `json:"position"`
		Kind         uint          `json:"kind"`
		PaddingLeft  bool          `json:"paddingLeft"`
		PaddingRight bool          `json:"paddingRight"`
	}

	InlayHintParams struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Range        Range                  `json:"range"`
	}

	SaveOptions struct {
		IncludeText bool `json:"includeText"`
	}

	DidSaveTextDocumentParams struct {
		Text         *string                `json:"text,omitempty"`
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}

	TextDocumentSyncOptions struct {
		Change    uint        `json:"change"`
		OpenClose bool        `json:"openClose"`
		Save      SaveOptions `json:"save"`
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
		TextDocument   TextDocumentIdentifier           `json:"textDocument"`
		ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
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

	MarkupContent struct {
		Kind  string `json:"kind"`
		Value string `json:"value"`
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

	Hover struct {
		Contents MarkupContent `json:"contents"`
		Range    Range         `json:"range"`
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

	SignatureHelpOptions struct {
		TriggerCharacters []string `json:"triggerCharacters,omitempty"`
	}

	SignatureHelpParams struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
		Position     Position               `json:"position"`
		Context      *SignatureHelpContext  `json:"context,omitempty"`
	}

	SignatureHelpContext struct {
		TriggerKind         uint           `json:"triggerKind"`
		TriggerCharacter    *string        `json:"triggerCharacter,omitempty"`
		IsRetrigger         bool           `json:"isRetrigger"`
		ActiveSignatureHelp *SignatureHelp `json:"activeSignatureHelp,omitempty"`
	}

	SignatureHelp struct {
		Signatures      []SignatureInformation `json:"signatures"`
		ActiveSignature *uint                  `json:"activeSignature,omitempty"`
		ActiveParameter *uint                  `json:"activeParameter,omitempty"`
	}

	SignatureInformation struct {
		Label           string                 `json:"label"`
		Documentation   string                 `json:"documentation,omitempty"`
		Parameters      []ParameterInformation `json:"parameters,omitempty"`
		ActiveParameter *uint                  `json:"activeParameter,omitempty"`
	}

	ParameterInformation struct {
		Label         string  `json:"label"`
		Documentation *string `json:"documentation,omitempty"`
	}

	iuint interface{ ~int | ~uint }
)
