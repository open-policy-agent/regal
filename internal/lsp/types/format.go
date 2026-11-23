// ComputeEdits is copied from https://github.com/kitagry/regols, the source repo's license is MIT and is copied below:
//
// MIT License
//
// # Copyright (c) 2023 Ryo Kitagawa
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package types

import (
	"strings"
)

// ComputeEdits computes diff edits from 2 string inputs.
func ComputeEdits(before, after string) []TextEdit {
	ops := operations(splitLines(before), splitLines(after))
	edits := make([]TextEdit, 0, len(ops))

	for _, op := range ops {
		switch op.Kind {
		case Delete:
			// Delete: unformatted[i1:i2] is deleted.
			edits = append(edits, TextEdit{Range: RangeBetween(op.I1, 0, op.I2, 0)})
		case Insert:
			// Insert: formatted[j1:j2] is inserted at unformatted[i1:i1].
			if content := strings.Join(op.Content, ""); content != "" {
				edits = append(edits, TextEdit{
					Range:   RangeBetween(op.I1, 0, op.I1, 0),
					NewText: content,
				})
			}
		case Equal:
		}
	}

	return edits
}
