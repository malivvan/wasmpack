package wasmpack

import (
	"bytes"
	"io"
	"strconv"
	"unicode"
)

// Minify takes a JavaScript source code as a byte slice and returns a minified version of it. It removes unnecessary
// whitespace and comments while preserving the functionality of the code.
func Minify(b []byte) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	prev := lineTerminatorToken
	prevLast := byte(' ')
	lineTerminatorQueued := false
	whitespaceQueued := false

	l := &jsLexer{
		stack:     make([]jsParsingContext, 0, 16),
		state:     exprState,
		emptyLine: true,
	}
	n := len(b)
	if n == 0 {
		l.buf = []byte{0}
	} else if b[n-1] != 0 {
		if cap(b) > n {
			b = b[:n+1]
			c := b[n]
			b[n] = 0
			l.buf = b
			l.restore = func() {
				b[n] = c
			}
		} else {
			l.buf = append(b, 0)
		}
	}

	defer func() {
		if l.restore != nil {
			l.restore()
			l.restore = nil
		}
	}()

	for {
		tt, data := l.next()
		if tt == errorToken {
			if l.Err() != io.EOF {
				return nil, l.Err()
			}
			return buf.Bytes(), nil
		} else if tt == lineTerminatorToken {
			lineTerminatorQueued = true
		} else if tt == whitespaceToken {
			whitespaceQueued = true
		} else if tt == singleLineCommentToken || tt == multiLineCommentToken {
			if len(data) > 5 && data[1] == '*' && data[2] == '!' {
				if _, err := buf.Write(data[:3]); err != nil {
					return nil, err
				}
				comment := replaceMultipleWhitespace(data[3 : len(data)-2])
				if tt != multiLineCommentToken {
					// don't trim newlines in multiline comments as that might change ASI
					// (we could do a more expensive check post-factum but it's not worth it)
					comment = trimWhitespace(comment)
				}
				if _, err := buf.Write(comment); err != nil {
					return nil, err
				}
				if _, err := buf.Write(data[len(data)-2:]); err != nil {
					return nil, err
				}
			} else if tt == multiLineCommentToken {
				lineTerminatorQueued = true
			} else {
				whitespaceQueued = true
			}
		} else {
			first := data[0]
			if (prev == identifierToken || prev == numericToken || prev == punctuatorToken || prev == stringToken || prev == templateToken || prev == regexpToken) &&
				(tt == identifierToken || tt == numericToken || tt == stringToken || tt == templateToken || tt == punctuatorToken || tt == regexpToken) {
				if lineTerminatorQueued && (prev != punctuatorToken || prevLast == '}' || prevLast == ']' || prevLast == ')' || prevLast == '+' || prevLast == '-' || prevLast == '"' || prevLast == '\'') &&
					(tt != punctuatorToken || first == '{' || first == '[' || first == '(' || first == '+' || first == '-' || first == '!' || first == '~') {
					if _, err := buf.Write([]byte("\n")); err != nil {
						return nil, err
					}
				} else if whitespaceQueued && (prev != stringToken && prev != punctuatorToken && tt != punctuatorToken || (prevLast == '+' || prevLast == '-' || prevLast == '/') && first == prevLast) {
					if _, err := buf.Write([]byte(" ")); err != nil {
						return nil, err
					}
				}
			}
			if _, err := buf.Write(data); err != nil {
				return nil, err
			}
			prev = tt
			prevLast = data[len(data)-1]
			lineTerminatorQueued = false
			whitespaceQueued = false
		}
	}
}

var jsIdentifierStart = []*unicode.RangeTable{unicode.Lu, unicode.Ll, unicode.Lt, unicode.Lm, unicode.Lo, unicode.Nl, unicode.Other_ID_Start}
var jsIdentifierContinue = []*unicode.RangeTable{unicode.Lu, unicode.Ll, unicode.Lt, unicode.Lm, unicode.Lo, unicode.Nl, unicode.Mn, unicode.Mc, unicode.Nd, unicode.Pc, unicode.Other_ID_Continue}

// jsTokenType determines the type of token, eg. a number or a semicolon.
type jsTokenType uint32

// jsTokenType values.
const (
	errorToken          jsTokenType = iota // extra token when errors occur
	unknownToken                           // extra token when no token can be matched
	whitespaceToken                        // space \t \v \f
	lineTerminatorToken                    // \r \n \r\n
	singleLineCommentToken
	multiLineCommentToken // token for comments with line terminators (not just any /*block*/)
	identifierToken
	punctuatorToken /* { } ( ) [ ] . ; , < > <= >= == != === !==  + - * % ++ -- << >>
	   >>> & | ^ ! ~ && || ? : = += -= *= %= <<= >>= >>>= &= |= ^= / /= >= */
	numericToken
	stringToken
	regexpToken
	templateToken
)

// jsTokenState determines a state in which next token should be read
type jsTokenState uint32

// jsTokenState values
const (
	exprState jsTokenState = iota
	stmtParensState
	subscriptState
	propNameState
)

// jsParsingContext determines the context in which following token should be parsed.
// jsThis affects parsing regular expressions and template literals.
type jsParsingContext uint32

// jsParsingContext values
const (
	globalContext jsParsingContext = iota
	stmtParensContext
	exprParensContext
	bracesContext
	templateContext
)

// String returns the string representation of a jsTokenType.
func (tt jsTokenType) String() string {
	switch tt {
	case errorToken:
		return "Error"
	case unknownToken:
		return "Unknown"
	case whitespaceToken:
		return "Whitespace"
	case lineTerminatorToken:
		return "LineTerminator"
	case singleLineCommentToken:
		return "SingleLineComment"
	case multiLineCommentToken:
		return "MultiLineComment"
	case identifierToken:
		return "Identifier"
	case punctuatorToken:
		return "Punctuator"
	case numericToken:
		return "Numeric"
	case stringToken:
		return "String"
	case regexpToken:
		return "Regexp"
	case templateToken:
		return "Template"
	}
	return "Invalid(" + strconv.Itoa(int(tt)) + ")"
}

// jsLexer is the state for the lexer.
type jsLexer struct {
	buf       []byte
	pos       int // index in buf
	start     int // index in buf
	err       error
	restore   func()
	stack     []jsParsingContext
	state     jsTokenState
	emptyLine bool
}

func (jsl *jsLexer) enterContext(context jsParsingContext) {
	jsl.stack = append(jsl.stack, context)
}

func (jsl *jsLexer) leaveContext() jsParsingContext {
	ctx := globalContext
	if last := len(jsl.stack) - 1; last >= 0 {
		ctx, jsl.stack = jsl.stack[last], jsl.stack[:last]
	}
	return ctx
}

// next returns the next Token. It returns errorToken when an error was encountered. Using Err() one can retrieve the error message.
func (jsl *jsLexer) next() (jsTokenType, []byte) {
	tt := unknownToken
	c := jsl.Peek(0)
	switch c {
	case '(':
		if jsl.state == stmtParensState {
			jsl.enterContext(stmtParensContext)
		} else {
			jsl.enterContext(exprParensContext)
		}
		jsl.state = exprState
		jsl.Move(1)
		tt = punctuatorToken
	case ')':
		if jsl.leaveContext() == stmtParensContext {
			jsl.state = exprState
		} else {
			jsl.state = subscriptState
		}
		jsl.Move(1)
		tt = punctuatorToken
	case '{':
		jsl.enterContext(bracesContext)
		jsl.state = exprState
		jsl.Move(1)
		tt = punctuatorToken
	case '}':
		if jsl.leaveContext() == templateContext && jsl.ConsumeTemplateToken() {
			tt = templateToken
		} else {
			// will work incorrectly for objects or functions divided by something,
			// but that's an extremely rare case
			jsl.state = exprState
			jsl.Move(1)
			tt = punctuatorToken
		}
	case ']':
		jsl.state = subscriptState
		jsl.Move(1)
		tt = punctuatorToken
	case '[', ';', ',', '~', '?', ':':
		jsl.state = exprState
		jsl.Move(1)
		tt = punctuatorToken
	case '<', '>', '=', '!', '+', '-', '*', '%', '&', '|', '^':
		if jsl.ConsumeHTMLLikeCommentToken() {
			return singleLineCommentToken, jsl.Shift()
		} else if jsl.ConsumeLongPunctuatorToken() {
			jsl.state = exprState
			tt = punctuatorToken
		}
	case '/':
		if tt = jsl.ConsumeCommentToken(); tt != unknownToken {
			return tt, jsl.Shift()
		} else if jsl.state == exprState && jsl.ConsumeRegexpToken() {
			jsl.state = subscriptState
			tt = regexpToken
		} else if jsl.ConsumeLongPunctuatorToken() {
			jsl.state = exprState
			tt = punctuatorToken
		}
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.':
		if jsl.ConsumeNumericToken() {
			tt = numericToken
			jsl.state = subscriptState
		} else if c == '.' {
			jsl.state = propNameState
			jsl.Move(1)
			tt = punctuatorToken
		}
	case '\'', '"':
		if jsl.ConsumeStringToken() {
			jsl.state = subscriptState
			tt = stringToken
		}
	case ' ', '\t', '\v', '\f':
		jsl.Move(1)
		for jsl.ConsumeWhitespace() {
		}
		return whitespaceToken, jsl.Shift()
	case '\n', '\r':
		jsl.Move(1)
		for jsl.ConsumeLineTerminator() {
		}
		tt = lineTerminatorToken
	case '`':
		if jsl.ConsumeTemplateToken() {
			tt = templateToken
		}
	default:
		if jsl.ConsumeIdentifierToken() {
			tt = identifierToken
			if jsl.state != propNameState {
				switch hash := toJsHash(jsl.Lexeme()); hash {
				case 0, jsThis, jsFalse, jsTrue, jsNull:
					jsl.state = subscriptState
				case jsIf, jsWhile, jsFor, jsWith:
					jsl.state = stmtParensState
				default:
					// jsThis will include keywords that can't be followed by a regexp, but only
					// by a specified char (like `switch` or `try`), but we don't check for syntax
					// errors as we don't attempt to parse a full JS grammar when streaming
					jsl.state = exprState
				}
			} else {
				jsl.state = subscriptState
			}
		} else if c >= 0xC0 {
			if jsl.ConsumeWhitespace() {
				for jsl.ConsumeWhitespace() {
				}
				return whitespaceToken, jsl.Shift()
			} else if jsl.ConsumeLineTerminator() {
				for jsl.ConsumeLineTerminator() {
				}
				tt = lineTerminatorToken
			}
		} else if jsl.Err() != nil {
			return errorToken, nil
		}
	}

	jsl.emptyLine = tt == lineTerminatorToken

	if tt == unknownToken {
		_, n := jsl.PeekRune(0)
		jsl.Move(n)
	}
	return tt, jsl.Shift()
}

/*
The following functions follow the specifications at http://www.ecma-international.org/ecma-262/5.1/
*/

func (jsl *jsLexer) ConsumeWhitespace() bool {
	c := jsl.Peek(0)
	if c == ' ' || c == '\t' || c == '\v' || c == '\f' {
		jsl.Move(1)
		return true
	} else if c >= 0xC0 {
		if r, n := jsl.PeekRune(0); r == '\u00A0' || r == '\uFEFF' || unicode.Is(unicode.Zs, r) {
			jsl.Move(n)
			return true
		}
	}
	return false
}

func (jsl *jsLexer) ConsumeLineTerminator() bool {
	c := jsl.Peek(0)
	if c == '\n' {
		jsl.Move(1)
		return true
	} else if c == '\r' {
		if jsl.Peek(1) == '\n' {
			jsl.Move(2)
		} else {
			jsl.Move(1)
		}
		return true
	} else if c >= 0xC0 {
		if r, n := jsl.PeekRune(0); r == '\u2028' || r == '\u2029' {
			jsl.Move(n)
			return true
		}
	}
	return false
}

func (jsl *jsLexer) ConsumeDigit() bool {
	if c := jsl.Peek(0); c >= '0' && c <= '9' {
		jsl.Move(1)
		return true
	}
	return false
}

func (jsl *jsLexer) ConsumeHexDigit() bool {
	if c := jsl.Peek(0); (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
		jsl.Move(1)
		return true
	}
	return false
}

func (jsl *jsLexer) ConsumeBinaryDigit() bool {
	if c := jsl.Peek(0); c == '0' || c == '1' {
		jsl.Move(1)
		return true
	}
	return false
}

func (jsl *jsLexer) ConsumeOctalDigit() bool {
	if c := jsl.Peek(0); c >= '0' && c <= '7' {
		jsl.Move(1)
		return true
	}
	return false
}

func (jsl *jsLexer) ConsumeUnicodeEscape() bool {
	if jsl.Peek(0) != '\\' || jsl.Peek(1) != 'u' {
		return false
	}
	mark := jsl.Pos()
	jsl.Move(2)
	if c := jsl.Peek(0); c == '{' {
		jsl.Move(1)
		if jsl.ConsumeHexDigit() {
			for jsl.ConsumeHexDigit() {
			}
			if c := jsl.Peek(0); c == '}' {
				jsl.Move(1)
				return true
			}
		}
		jsl.Rewind(mark)
		return false
	} else if !jsl.ConsumeHexDigit() || !jsl.ConsumeHexDigit() || !jsl.ConsumeHexDigit() || !jsl.ConsumeHexDigit() {
		jsl.Rewind(mark)
		return false
	}
	return true
}

func (jsl *jsLexer) ConsumeSingleLineComment() {
	for {
		c := jsl.Peek(0)
		if c == '\r' || c == '\n' || c == 0 {
			break
		} else if c >= 0xC0 {
			if r, _ := jsl.PeekRune(0); r == '\u2028' || r == '\u2029' {
				break
			}
		}
		jsl.Move(1)
	}
}

func (jsl *jsLexer) ConsumeHTMLLikeCommentToken() bool {
	c := jsl.Peek(0)
	if c == '<' && jsl.Peek(1) == '!' && jsl.Peek(2) == '-' && jsl.Peek(3) == '-' {
		// opening HTML-style single line comment
		jsl.Move(4)
		jsl.ConsumeSingleLineComment()
		return true
	} else if jsl.emptyLine && c == '-' && jsl.Peek(1) == '-' && jsl.Peek(2) == '>' {
		// closing HTML-style single line comment
		// (only if current line didn't contain any meaningful tokens)
		jsl.Move(3)
		jsl.ConsumeSingleLineComment()
		return true
	}
	return false
}

func (jsl *jsLexer) ConsumeCommentToken() jsTokenType {
	c := jsl.Peek(0)
	if c == '/' {
		c = jsl.Peek(1)
		if c == '/' {
			// single line comment
			jsl.Move(2)
			jsl.ConsumeSingleLineComment()
			return singleLineCommentToken
		} else if c == '*' {
			// block comment (potentially multiline)
			tt := singleLineCommentToken
			jsl.Move(2)
			for {
				c := jsl.Peek(0)
				if c == '*' && jsl.Peek(1) == '/' {
					jsl.Move(2)
					break
				} else if c == 0 {
					break
				} else if jsl.ConsumeLineTerminator() {
					tt = multiLineCommentToken
					jsl.emptyLine = true
				} else {
					jsl.Move(1)
				}
			}
			return tt
		}
	}
	return unknownToken
}

func (jsl *jsLexer) ConsumeLongPunctuatorToken() bool {
	c := jsl.Peek(0)
	if c == '!' || c == '=' || c == '+' || c == '-' || c == '*' || c == '/' || c == '%' || c == '&' || c == '|' || c == '^' {
		jsl.Move(1)
		if jsl.Peek(0) == '=' {
			jsl.Move(1)
			if (c == '!' || c == '=') && jsl.Peek(0) == '=' {
				jsl.Move(1)
			}
		} else if (c == '+' || c == '-' || c == '&' || c == '|') && jsl.Peek(0) == c {
			jsl.Move(1)
		} else if c == '=' && jsl.Peek(0) == '>' {
			jsl.Move(1)
		}
	} else { // c == '<' || c == '>'
		jsl.Move(1)
		if jsl.Peek(0) == c {
			jsl.Move(1)
			if c == '>' && jsl.Peek(0) == '>' {
				jsl.Move(1)
			}
		}
		if jsl.Peek(0) == '=' {
			jsl.Move(1)
		}
	}
	return true
}

func (jsl *jsLexer) ConsumeIdentifierToken() bool {
	c := jsl.Peek(0)
	if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '$' || c == '_' {
		jsl.Move(1)
	} else if c >= 0xC0 {
		if r, n := jsl.PeekRune(0); unicode.IsOneOf(jsIdentifierStart, r) {
			jsl.Move(n)
		} else {
			return false
		}
	} else if !jsl.ConsumeUnicodeEscape() {
		return false
	}
	for {
		c := jsl.Peek(0)
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '$' || c == '_' {
			jsl.Move(1)
		} else if c >= 0xC0 {
			if r, n := jsl.PeekRune(0); r == '\u200C' || r == '\u200D' || unicode.IsOneOf(jsIdentifierContinue, r) {
				jsl.Move(n)
			} else {
				break
			}
		} else {
			break
		}
	}
	return true
}

func (jsl *jsLexer) ConsumeNumericToken() bool {
	// assume to be on 0 1 2 3 4 5 6 7 8 9 .
	mark := jsl.Pos()
	c := jsl.Peek(0)
	if c == '0' {
		jsl.Move(1)
		if jsl.Peek(0) == 'x' || jsl.Peek(0) == 'X' {
			jsl.Move(1)
			if jsl.ConsumeHexDigit() {
				for jsl.ConsumeHexDigit() {
				}
			} else {
				jsl.Move(-1) // return just the zero
			}
			return true
		} else if jsl.Peek(0) == 'b' || jsl.Peek(0) == 'B' {
			jsl.Move(1)
			if jsl.ConsumeBinaryDigit() {
				for jsl.ConsumeBinaryDigit() {
				}
			} else {
				jsl.Move(-1) // return just the zero
			}
			return true
		} else if jsl.Peek(0) == 'o' || jsl.Peek(0) == 'O' {
			jsl.Move(1)
			if jsl.ConsumeOctalDigit() {
				for jsl.ConsumeOctalDigit() {
				}
			} else {
				jsl.Move(-1) // return just the zero
			}
			return true
		}
	} else if c != '.' {
		for jsl.ConsumeDigit() {
		}
	}
	if jsl.Peek(0) == '.' {
		jsl.Move(1)
		if jsl.ConsumeDigit() {
			for jsl.ConsumeDigit() {
			}
		} else if c != '.' {
			// . could belong to the next token
			jsl.Move(-1)
			return true
		} else {
			jsl.Rewind(mark)
			return false
		}
	}
	mark = jsl.Pos()
	c = jsl.Peek(0)
	if c == 'e' || c == 'E' {
		jsl.Move(1)
		c = jsl.Peek(0)
		if c == '+' || c == '-' {
			jsl.Move(1)
		}
		if !jsl.ConsumeDigit() {
			// e could belong to the next token
			jsl.Rewind(mark)
			return true
		}
		for jsl.ConsumeDigit() {
		}
	}
	return true
}

func (jsl *jsLexer) ConsumeStringToken() bool {
	// assume to be on ' or "
	mark := jsl.Pos()
	delim := jsl.Peek(0)
	jsl.Move(1)
	for {
		c := jsl.Peek(0)
		if c == delim {
			jsl.Move(1)
			break
		} else if c == '\\' {
			jsl.Move(1)
			if !jsl.ConsumeLineTerminator() {
				if c := jsl.Peek(0); c == delim || c == '\\' {
					jsl.Move(1)
				}
			}
			continue
		} else if c == '\n' || c == '\r' {
			jsl.Rewind(mark)
			return false
		} else if c >= 0xC0 {
			if r, _ := jsl.PeekRune(0); r == '\u2028' || r == '\u2029' {
				jsl.Rewind(mark)
				return false
			}
		} else if c == 0 {
			break
		}
		jsl.Move(1)
	}
	return true
}

func (jsl *jsLexer) ConsumeRegexpToken() bool {
	// assume to be on / and not /*
	mark := jsl.Pos()
	jsl.Move(1)
	inClass := false
	for {
		c := jsl.Peek(0)
		if !inClass && c == '/' {
			jsl.Move(1)
			break
		} else if c == '[' {
			inClass = true
		} else if c == ']' {
			inClass = false
		} else if c == '\\' {
			jsl.Move(1)
			if jsl.ConsumeLineTerminator() {
				jsl.Rewind(mark)
				return false
			} else if jsl.Peek(0) == 0 {
				return true
			}
		} else if jsl.ConsumeLineTerminator() {
			jsl.Rewind(mark)
			return false
		} else if c == 0 {
			return true
		}
		jsl.Move(1)
	}
	// flags
	for {
		c := jsl.Peek(0)
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '$' || c == '_' {
			jsl.Move(1)
		} else if c >= 0xC0 {
			if r, n := jsl.PeekRune(0); r == '\u200C' || r == '\u200D' || unicode.IsOneOf(jsIdentifierContinue, r) {
				jsl.Move(n)
			} else {
				break
			}
		} else {
			break
		}
	}
	return true
}

func (jsl *jsLexer) ConsumeTemplateToken() bool {
	// assume to be on ` or } when already within template
	mark := jsl.Pos()
	jsl.Move(1)
	for {
		c := jsl.Peek(0)
		if c == '`' {
			jsl.state = subscriptState
			jsl.Move(1)
			return true
		} else if c == '$' && jsl.Peek(1) == '{' {
			jsl.enterContext(templateContext)
			jsl.state = exprState
			jsl.Move(2)
			return true
		} else if c == '\\' {
			jsl.Move(1)
			if c := jsl.Peek(0); c != 0 {
				jsl.Move(1)
			}
			continue
		} else if c == 0 {
			jsl.Rewind(mark)
			return false
		}
		jsl.Move(1)
	}
}

// Restore restores the replaced byte past the end of the buffer by NULL.
func (jsl *jsLexer) Restore() {
	if jsl.restore != nil {
		jsl.restore()
		jsl.restore = nil
	}
}

// Err returns the error returned from io.Reader or io.EOF when the end has been reached.
func (jsl *jsLexer) Err() error {
	return jsl.PeekErr(0)
}

// PeekErr returns the error at position pos. When pos is zero, this is the same as calling Err().
func (jsl *jsLexer) PeekErr(pos int) error {
	if jsl.err != nil {
		return jsl.err
	} else if jsl.pos+pos >= len(jsl.buf)-1 {
		return io.EOF
	}
	return nil
}

// Peek returns the ith byte relative to the end position.
// Peek returns 0 when an error has occurred, Err returns the error.
func (jsl *jsLexer) Peek(pos int) byte {
	pos += jsl.pos
	return jsl.buf[pos]
}

// PeekRune returns the rune and rune length of the ith byte relative to the end position.
func (jsl *jsLexer) PeekRune(pos int) (rune, int) {
	// from unicode/utf8
	c := jsl.Peek(pos)
	if c < 0xC0 || jsl.Peek(pos+1) == 0 {
		return rune(c), 1
	} else if c < 0xE0 || jsl.Peek(pos+2) == 0 {
		return rune(c&0x1F)<<6 | rune(jsl.Peek(pos+1)&0x3F), 2
	} else if c < 0xF0 || jsl.Peek(pos+3) == 0 {
		return rune(c&0x0F)<<12 | rune(jsl.Peek(pos+1)&0x3F)<<6 | rune(jsl.Peek(pos+2)&0x3F), 3
	}
	return rune(c&0x07)<<18 | rune(jsl.Peek(pos+1)&0x3F)<<12 | rune(jsl.Peek(pos+2)&0x3F)<<6 | rune(jsl.Peek(pos+3)&0x3F), 4
}

// Move advances the position.
func (jsl *jsLexer) Move(n int) {
	jsl.pos += n
}

// Pos returns a mark to which can be rewinded.
func (jsl *jsLexer) Pos() int {
	return jsl.pos - jsl.start
}

// Rewind rewinds the position to the given position.
func (jsl *jsLexer) Rewind(pos int) {
	jsl.pos = jsl.start + pos
}

// Lexeme returns the bytes of the current selection.
func (jsl *jsLexer) Lexeme() []byte {
	return jsl.buf[jsl.start:jsl.pos]
}

// Skip collapses the position to the end of the selection.
func (jsl *jsLexer) Skip() {
	jsl.start = jsl.pos
}

// Shift returns the bytes of the current selection and collapses the position to the end of the selection.
func (jsl *jsLexer) Shift() []byte {
	b := jsl.buf[jsl.start:jsl.pos]
	jsl.start = jsl.pos
	return b
}

// Offset returns the character position in the buffer.
func (jsl *jsLexer) Offset() int {
	return jsl.pos
}

// Bytes returns the underlying buffer.
func (jsl *jsLexer) Bytes() []byte {
	return jsl.buf
}

// jsHash defines perfect hashes for a predefined list of strings
type jsHash uint32

// Unique hash definitions to be used instead of strings
const (
	jsBreak      jsHash = 0x5    // break
	jsCase       jsHash = 0x3404 // case
	jsCatch      jsHash = 0xba05 // catch
	jsClass      jsHash = 0x505  // class
	jsConst      jsHash = 0x2c05 // const
	jsContinue   jsHash = 0x3e08 // continue
	jsDebugger   jsHash = 0x8408 // debugger
	jsDefault    jsHash = 0xab07 // default
	jsDelete     jsHash = 0xcd06 // delete
	jsDo         jsHash = 0x4c02 // do
	jsElse       jsHash = 0x3704 // else
	jsEnum       jsHash = 0x3a04 // enum
	jsExport     jsHash = 0x1806 // export
	jsExtends    jsHash = 0x4507 // extends
	jsFalse      jsHash = 0x5a05 // false
	jsFinally    jsHash = 0x7a07 // finally
	jsFor        jsHash = 0xc403 // for
	jsFunction   jsHash = 0x4e08 // function
	jsIf         jsHash = 0x5902 // if
	jsImplements jsHash = 0x5f0a // implements
	jsImport     jsHash = 0x6906 // import
	jsIn         jsHash = 0x4202 // in
	jsInstanceof jsHash = 0x710a // instanceof
	jsInterface  jsHash = 0x8c09 // interface
	jsLet        jsHash = 0xcf03 // let
	jsNew        jsHash = 0x1203 // new
	jsNull       jsHash = 0x5504 // null
	jsPackage    jsHash = 0x9507 // package
	jsPrivate    jsHash = 0x9c07 // private
	jsProtected  jsHash = 0xa309 // protected
	jsPublic     jsHash = 0xb506 // public
	jsReturn     jsHash = 0xd06  // return
	jsStatic     jsHash = 0x2f06 // static
	jsSuper      jsHash = 0x905  // super
	jsSwitch     jsHash = 0x2606 // switch
	jsThis       jsHash = 0x2304 // this
	jsThrow      jsHash = 0x1d05 // throw
	jsTrue       jsHash = 0xb104 // true
	jsTry        jsHash = 0x6e03 // try
	jsTypeof     jsHash = 0xbf06 // typeof
	jsVar        jsHash = 0xc703 // var
	jsVoid       jsHash = 0xca04 // void
	jsWhile      jsHash = 0x1405 // while
	jsWith       jsHash = 0x2104 // with
	jsYield      jsHash = 0x8005 // yield
)

// String returns the hash' name.
func (i jsHash) String() string {
	start := uint32(i >> 8)
	n := uint32(i & 0xff)
	if start+n > uint32(len(hashText)) {
		return ""
	}
	return hashText[start : start+n]
}

// toJsHash returns the hash whose name is s. It returns zero if there is no
// such hash. It is case sensitive.
func toJsHash(s []byte) jsHash {
	if len(s) == 0 || len(s) > hashMaxlen {
		return 0
	}
	h := uint32(hashHash0)
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	if i := hashTable[h&uint32(len(hashTable)-1)]; int(i&0xff) == len(s) {
		t := hashText[i>>8 : i>>8+i&0xff]
		for i := 0; i < len(s); i++ {
			if t[i] != s[i] {
				goto NEXT
			}
		}
		return i
	}
NEXT:
	if i := hashTable[(h>>16)&uint32(len(hashTable)-1)]; int(i&0xff) == len(s) {
		t := hashText[i>>8 : i>>8+i&0xff]
		for i := 0; i < len(s); i++ {
			if t[i] != s[i] {
				return 0
			}
		}
		return i
	}
	return 0
}

const hashHash0 = 0x9acb0442
const hashMaxlen = 10
const hashText = "breakclassupereturnewhilexporthrowithiswitchconstaticaselsen" +
	"umcontinuextendsdofunctionullifalseimplementsimportryinstanc" +
	"eofinallyieldebuggerinterfacepackageprivateprotectedefaultru" +
	"epublicatchtypeoforvarvoidelete"

var hashTable = [1 << 6]jsHash{
	0x0:  0x2f06, // static
	0x1:  0x9c07, // private
	0x3:  0xb104, // true
	0x6:  0x5a05, // false
	0x7:  0x4c02, // do
	0x9:  0x2c05, // const
	0xa:  0x2606, // switch
	0xb:  0x6e03, // try
	0xc:  0x1203, // new
	0xd:  0x4202, // in
	0xf:  0x8005, // yield
	0x10: 0x5f0a, // implements
	0x11: 0xc403, // for
	0x12: 0x505,  // class
	0x13: 0x3a04, // enum
	0x16: 0xc703, // var
	0x17: 0x5902, // if
	0x19: 0xcf03, // let
	0x1a: 0x9507, // package
	0x1b: 0xca04, // void
	0x1c: 0xcd06, // delete
	0x1f: 0x5504, // null
	0x20: 0x1806, // export
	0x21: 0xd06,  // return
	0x23: 0x4507, // extends
	0x25: 0x2304, // this
	0x26: 0x905,  // super
	0x27: 0x1405, // while
	0x29: 0x5,    // break
	0x2b: 0x3e08, // continue
	0x2e: 0x3404, // case
	0x2f: 0xab07, // default
	0x31: 0x8408, // debugger
	0x32: 0x1d05, // throw
	0x33: 0xbf06, // typeof
	0x34: 0x2104, // with
	0x35: 0xba05, // catch
	0x36: 0x4e08, // function
	0x37: 0x710a, // instanceof
	0x38: 0xa309, // protected
	0x39: 0x8c09, // interface
	0x3b: 0xb506, // public
	0x3c: 0x3704, // else
	0x3d: 0x7a07, // finally
	0x3f: 0x6906, // import
}

var whitespaceTable = [256]bool{
	// ASCII
	false, false, false, false, false, false, false, false,
	false, true, true, false, true, true, false, false, // tab, new line, form feed, carriage return
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	true, false, false, false, false, false, false, false, // space
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	// non-ASCII
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
}

// isWhitespace returns true for space, \n, \r, \t, \f.
func isWhitespace(c byte) bool {
	return whitespaceTable[c]
}

var newlineTable = [256]bool{
	// ASCII
	false, false, false, false, false, false, false, false,
	false, false, true, false, false, true, false, false, // new line, carriage return
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	// non-ASCII
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,

	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
	false, false, false, false, false, false, false, false,
}

// isNewline returns true for \n, \r.
func isNewline(c byte) bool {
	return newlineTable[c]
}

// trimWhitespace removes any leading and trailing whitespace characters.
func trimWhitespace(b []byte) []byte {
	n := len(b)
	start := n
	for i := 0; i < n; i++ {
		if !isWhitespace(b[i]) {
			start = i
			break
		}
	}
	end := n
	for i := n - 1; i >= start; i-- {
		if !isWhitespace(b[i]) {
			end = i + 1
			break
		}
	}
	return b[start:end]
}

// replaceMultipleWhitespace replaces character series of space, \n, \t, \f, \r into a single space or newline (when the serie contained a \n or \r).
func replaceMultipleWhitespace(b []byte) []byte {
	j := 0
	prevWS := false
	hasNewline := false
	for i, c := range b {
		if isWhitespace(c) {
			prevWS = true
			if isNewline(c) {
				hasNewline = true
			}
		} else {
			if prevWS {
				prevWS = false
				if hasNewline {
					hasNewline = false
					b[j] = '\n'
				} else {
					b[j] = ' '
				}
				j++
			}
			b[j] = b[i]
			j++
		}
	}
	if prevWS {
		if hasNewline {
			b[j] = '\n'
		} else {
			b[j] = ' '
		}
		j++
	}
	return b[:j]
}
