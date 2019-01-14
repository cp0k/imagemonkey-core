package imagemonkeyquerylang

import (
	"errors"
	"github.com/antlr/antlr4/runtime/Go/antlr"
	"strconv"
	"strings"
)

type CustomErrorListener struct {
	*antlr.DefaultErrorListener
	err error
	query string
}

func NewCustomErrorListener() *CustomErrorListener {
	return &CustomErrorListener {
        err: nil,
        query: "",
    } 
}

func (c *CustomErrorListener) SyntaxError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string, e antlr.RecognitionException) {
	//c.err = errors.New(msg)
	c. err = errors.New(c.underlineError(recognizer, offendingSymbol, line, column, msg))
}

func (c *CustomErrorListener) underlineError(recognizer antlr.Recognizer, offendingSymbol interface{}, line, column int, msg string) string {
	out := c.query + "\n"
	for i := 0; i < column; i++ {
		out += " "
	}
	start := offendingSymbol.(antlr.Token).GetStart()
	stop := offendingSymbol.(antlr.Token).GetStop()
	if start >= 0 && stop >= 0 {
		for i := start; i <= stop; i++ {
			out += "^"
		}
	}
	out += "\n" + msg
	return out
}


type imagemonkeyQueryLangListener struct {
	*BaseImagemonkeyQueryLangListener
	pos int
	allowStaticQueryAttributes bool
	numOfLabels int
	version int
	typeOfQueryKnown bool
	err error
	isUuidQuery bool

	stack []ParseResult

}

type StackEntry struct {
	Val string
}


func containsOnly(s string, c rune) bool {
	for _, char := range s {
		if char != c {
			return false
		}
	}
	return true
}

func (l *imagemonkeyQueryLangListener) push(r ParseResult) {
	l.stack = append(l.stack, r)
}

func (l *imagemonkeyQueryLangListener) pop() ParseResult {
	if len(l.stack) < 1 {
		return ParseResult{}
	}

	// Get the last value from the stack.
	result := l.stack[len(l.stack) - 1]

	// Remove the last element from the stack.
	l.stack = l.stack[:len(l.stack) - 1]

	return result
}

func (l *imagemonkeyQueryLangListener) peek() ParseResult {
	if len(l.stack) < 1 {
		return ParseResult{}
	}

	// Get the last value from the stack.
	result := l.stack[len(l.stack) - 1]

	return result
}

func (l *imagemonkeyQueryLangListener) popParentheses() string {
	prefix := ""

	for {
		if len(l.stack) > 0 {
			res := l.peek()
			if containsOnly(res.Query, '(') {
				prefix += res.Query
				l.pop()
			} else {
				break
			}
		} else {
			break
		}
	}

	return prefix
}


func (l *imagemonkeyQueryLangListener) EnterParenthesesExpression(c *ParenthesesExpressionContext) {
	stackEntry := ParseResult{Query: "("}
	l.push(stackEntry)
}

func (l *imagemonkeyQueryLangListener) ExitParenthesesExpression(c *ParenthesesExpressionContext) {
	if len(l.stack) > 0 {
		stackEntry := l.pop()
		stackEntry.Query = stackEntry.Query + ")"

		//stackEntry.Subquery = stackEntry.Subquery + ")"
		l.push(stackEntry)
	}
}

func (l *imagemonkeyQueryLangListener) ExitAnnotationCoverageExpression(c *AnnotationCoverageExpressionContext) {
	if l.allowStaticQueryAttributes {
		tokens := c.GetTokens(ImagemonkeyQueryLangParserVAL)
		if len(tokens) > 0 {
			annotationCoverageVal := tokens[0].GetText()
			if _, err := strconv.Atoi(annotationCoverageVal); err == nil {
				tokens = c.GetTokens(ImagemonkeyQueryLangParserOPERATOR)
				if len(tokens) > 0 {
					operator := tokens[0].GetText()
					val := "q.annotated_percentage" + operator + annotationCoverageVal

					stackEntry := ParseResult{Query: val}
					//stackEntry.Subquery = val
					l.push(stackEntry)
				}
			}
		}
	}
}

func (l *imagemonkeyQueryLangListener) ExitImageWidthExpression(c *ImageWidthExpressionContext) {
	if l.allowStaticQueryAttributes {
		tokens := c.GetTokens(ImagemonkeyQueryLangParserVAL)
		if len(tokens) > 0 {
			imageWidthVal := tokens[0].GetText()
			if _, err := strconv.Atoi(imageWidthVal); err == nil {
				tokens = c.GetTokens(ImagemonkeyQueryLangParserOPERATOR)
				if len(tokens) > 0 {
					operator := tokens[0].GetText()
					val := "image_width" + operator + imageWidthVal

					stackEntry := ParseResult{Query: val}
					//stackEntry.Subquery = val
					l.push(stackEntry)
				}
			}
		}
	}
}

func (l *imagemonkeyQueryLangListener) ExitImageHeightExpression(c *ImageHeightExpressionContext) {
	if l.allowStaticQueryAttributes {
		tokens := c.GetTokens(ImagemonkeyQueryLangParserVAL)
		if len(tokens) > 0 {
			imageHeightVal := tokens[0].GetText()
			if _, err := strconv.Atoi(imageHeightVal); err == nil {
				tokens = c.GetTokens(ImagemonkeyQueryLangParserOPERATOR)
				if len(tokens) > 0 {
					operator := tokens[0].GetText()
					val := "q.image_height" + operator + imageHeightVal

					stackEntry := ParseResult{Query: val}
					//stackEntry.Subquery = val
					l.push(stackEntry)
				}
			}
		}
	}
}

func (l *imagemonkeyQueryLangListener) ExitUuidExpression(c *UuidExpressionContext) {
	var stackEntry ParseResult

	//the first token determines if it's a UUID query or not
	if !l.typeOfQueryKnown {
		l.typeOfQueryKnown = true
		l.isUuidQuery = true
	}

	val := ""
	if l.version == 1 {
		if !l.isUuidQuery {
			l.err = errors.New("Expecting UUID, got " +strings.TrimSpace(c.GetText()))
			return
		}
		val = "a.accessor = $" + strconv.Itoa(l.pos)
	} else {
		l.err = errors.New("UUID not allowed")
	}

	stackEntry = ParseResult{Query: val}
	stackEntry.QueryValues = append(stackEntry.QueryValues, strings.TrimSpace(c.GetText())) //remove leading + trailing spaces
	stackEntry.Subquery = val
	l.push(stackEntry)

	l.pos += 1
	l.numOfLabels += 1
}

func (l *imagemonkeyQueryLangListener) ExitLabelExpression(c *LabelExpressionContext) {
	//the first token determines if it's a UUID query or not
	if !l.typeOfQueryKnown {
		l.typeOfQueryKnown = true
		l.isUuidQuery = false
	}

	val := ""
	if l.version == 1 {
		if l.isUuidQuery {
			l.err = errors.New("Expecting label, got " + strings.TrimSpace(c.GetText()))
			return
		}
		val = "a.accessor = $" + strconv.Itoa(l.pos)
	} else {
		val = "q.accessors @> ARRAY[$" + strconv.Itoa(l.pos) + "]::text[]"
	}

	
	subval := "a.accessor = $" + strconv.Itoa(l.pos)

	stackEntry := ParseResult{Query: val}
	stackEntry.QueryValues = append(stackEntry.QueryValues, strings.TrimSpace(c.GetText())) //remove leading + trailing spaces
	stackEntry.Subquery = subval
	l.push(stackEntry)

	l.pos += 1
	l.numOfLabels += 1
}

func (l *imagemonkeyQueryLangListener) ExitAssignmentExpression(c *AssignmentExpressionContext) {
	//the first token determines if it's a UUID query or not
	if !l.typeOfQueryKnown {
		l.typeOfQueryKnown = true
		l.isUuidQuery = false
	}

	val := ""
	if l.version == 1 {
		if l.isUuidQuery {
			l.err = errors.New("Expecting UUID, got " + c.GetText())
			return
		}
		val = "a.accessor = $" + strconv.Itoa(l.pos)
	} else {
		val = "q.accessors @> ARRAY[$" + strconv.Itoa(l.pos) + "]::text[]"
	}

	assignmentVal := strings.TrimSpace(c.GetText()) //remove leading + trailing spaces

	equalSignPos := strings.Index(assignmentVal, "=")
	if equalSignPos != - 1 { //found
		beforePart := strings.TrimSpace(assignmentVal[: equalSignPos])
		afterPart := strings.TrimSpace(assignmentVal[equalSignPos + 1 :])

		assignmentVal = beforePart + "=" + afterPart

		stackEntry := ParseResult{Query: val}
		stackEntry.QueryValues = append(stackEntry.QueryValues, assignmentVal)
		l.push(stackEntry)

		l.pos += 1
		l.numOfLabels += 1

	}
}

func (l *imagemonkeyQueryLangListener) ExitAndExpression(c *AndExpressionContext) {
	right := l.pop()
	rightParentheses := l.popParentheses()

	left := l.pop()
	leftParentheses := l.popParentheses()
	
	stackEntry := ParseResult{Query: leftParentheses + left.Query + " AND " + rightParentheses + right.Query}
	if left.Subquery != "" && right.Subquery != "" {
		stackEntry.Subquery = leftParentheses + left.Subquery + " OR " + rightParentheses + right.Subquery
	} else {
		if left.Subquery != "" {
			stackEntry.Subquery = left.Subquery
		} else if right.Subquery != "" {
			stackEntry.Subquery = right.Subquery
		}
	}
	stackEntry.QueryValues = append(stackEntry.QueryValues, append(left.QueryValues, right.QueryValues...)...)
	l.push(stackEntry) 
}

func (l *imagemonkeyQueryLangListener) ExitNotExpression(c *NotExpressionContext) {
	left := l.pop()
	
	stackEntry := ParseResult{Query: "NOT " + left.Query}
	if left.Subquery != "" {
		stackEntry.Subquery = "NOT " + left.Subquery
	}
	stackEntry.QueryValues = append(stackEntry.QueryValues, left.QueryValues...)
	l.push(stackEntry)
}

func (l *imagemonkeyQueryLangListener) ExitOrExpression(c *OrExpressionContext) {
	right := l.pop()
	rightParentheses := l.popParentheses()

	left := l.pop()
	leftParentheses := l.popParentheses()

	stackEntry := ParseResult{Query: leftParentheses + left.Query + " OR " + rightParentheses + right.Query}
	if left.Subquery != "" && right.Subquery != "" {
		stackEntry.Subquery = leftParentheses + left.Subquery + " OR " + rightParentheses + right.Subquery
	} else {
		if left.Subquery != "" {
			stackEntry.Subquery = left.Subquery
		} else if right.Subquery != "" {
			stackEntry.Subquery = right.Subquery
		}
	}
	stackEntry.QueryValues = append(stackEntry.QueryValues, append(left.QueryValues, right.QueryValues...)...)
	l.push(stackEntry) 
}