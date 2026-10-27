package params

import (
	"github.com/open-policy-agent/opa/v1/ast"

	"github.com/open-policy-agent/regal/pkg/roast/rast"
)

type Params struct {
	DisableCategory []string `json:"disable_category"`
	Disable         []string `json:"disable"`
	EnableCategory  []string `json:"enable_category"`
	Enable          []string `json:"enable"`
	IgnoreFiles     []string `json:"ignore_files"`
	DisableAll      bool     `json:"disable_all"`
	EnableAll       bool     `json:"enable_all"`
}

var (
	None     = &Params{}
	NoneTerm *ast.Term
)

func init() {
	ast.InternStringTerm(
		"disable_all", "disable_category", "disable", "enable_all", "enable_category", "enable", "ignore_files",
	)

	NoneTerm = ast.ObjectTerm(
		rast.Item("disable_all", ast.InternedBooleanFalse),
		rast.Item("disable_category", ast.InternedEmptyArray),
		rast.Item("disable", ast.InternedEmptyArray),
		rast.Item("enable_all", ast.InternedBooleanFalse),
		rast.Item("enable_category", ast.InternedEmptyArray),
		rast.Item("enable", ast.InternedEmptyArray),
		rast.Item("ignore_files", ast.InternedEmptyArray),
	)
}

func New() *Params {
	return &Params{}
}

func (p *Params) WithDisableAll(disableAll bool) *Params {
	p.DisableAll = disableAll

	return p
}

func (p *Params) WithEnableAll(enableAll bool) *Params {
	p.EnableAll = enableAll

	return p
}

func (p *Params) WithDisableCategory(disableCategory ...string) *Params {
	p.DisableCategory = disableCategory

	return p
}

func (p *Params) WithDisable(disable ...string) *Params {
	p.Disable = disable

	return p
}

func (p *Params) WithEnableCategory(enableCategory ...string) *Params {
	p.EnableCategory = enableCategory

	return p
}

func (p *Params) WithEnable(enable ...string) *Params {
	p.Enable = enable

	return p
}

func (p *Params) WithIgnoreFiles(ignoreFiles ...string) *Params {
	p.IgnoreFiles = ignoreFiles

	return p
}

func (p *Params) IsNone() bool {
	return p == nil || p == None || (!p.DisableAll && !p.EnableAll &&
		len(p.DisableCategory) == 0 && len(p.Disable) == 0 &&
		len(p.EnableCategory) == 0 && len(p.Enable) == 0 && len(p.IgnoreFiles) == 0)
}

func (p *Params) ToTerm() *ast.Term {
	if p.IsNone() {
		return NoneTerm
	}

	return ast.ObjectTerm(
		rast.Item("disable_all", ast.InternedTerm(p.DisableAll)),
		rast.Item("disable_category", rast.ArrayTerm(p.DisableCategory...)),
		rast.Item("disable", rast.ArrayTerm(p.Disable...)),
		rast.Item("enable_all", ast.InternedTerm(p.EnableAll)),
		rast.Item("enable_category", rast.ArrayTerm(p.EnableCategory...)),
		rast.Item("enable", rast.ArrayTerm(p.Enable...)),
		rast.Item("ignore_files", rast.ArrayTerm(p.IgnoreFiles...)),
	)
}
