# METADATA
# description: support functions to render example links in hover tooltips
package regal.lsp.hover.example

# METADATA
# description: return rendered hover information for built-in functions
# scope: document
default builtin(_) := ""

builtin(name) := $"[View Usage Examples]({_base}/builtins/{path})\n\n" if path := _paths.builtins[name]

# METADATA
# description: return rendered hover information for keyword of name
keyword(name) := $"[View Usage Examples]({_base}/keywords/{name})\n\n"

_base := "https://www.openpolicyagent.org/docs/policy-reference/"

_paths["builtins"] := {
	"contains": "strings#contains",
	"io.jwt.decode_verify": "tokens#decode_verify",
	"io.jwt.verify_es256": "tokens#verify_es256",
	"io.jwt.encode_sign": "tokensign#encode_sign",
	"io.jwt.encode_sign_raw": "tokensign#encode_sign_raw",
	"crypto.md5": "crypto#cryptomd5",
	"glob.match": "glob#globmatch",
	"print": "opa#print-function",
	"regex.match": "regex#match",
	"regex.template_match": "regex#template_match",
	"find_all_string_submatch_n": "regex#find_all_string_submatch_n",
	"regex.globs_match": "regex#globs_match",
	"time.clock": "time#clock",
	"time.format": "time#format",
	"time.parse_ns": "time#parse_ns",
	"time.now_ns": "time#now_ns",
}
