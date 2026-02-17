package backend

import (
	"fmt"
	"strings"

	rdiff "robot_diff/backend/diff"
)

func buildTestBodyKeywords(test *rdiff.Test) []rdiff.Keyword {
	if len(test.Body) > 0 {
		return orderedBodyToKeywords(test.Body)
	}
	keywords := make([]rdiff.Keyword, 0, len(test.Keywords)+len(test.Ifs)+len(test.Fors))
	keywords = append(keywords, test.Keywords...)
	keywords = append(keywords, controlFlowToKeywords(test.Ifs, test.Fors)...)
	return keywords
}

func orderedBodyToKeywords(body []rdiff.BodyItem) []rdiff.Keyword {
	out := make([]rdiff.Keyword, 0, len(body))
	for _, it := range body {
		switch {
		case it.Keyword != nil:
			out = append(out, *it.Keyword)
		case it.If != nil:
			out = append(out, ifToKeyword(*it.If))
		case it.For != nil:
			out = append(out, forToKeyword(*it.For))
		}
	}
	return out
}

func controlFlowToKeywords(ifs []rdiff.If, fors []rdiff.For) []rdiff.Keyword {
	out := make([]rdiff.Keyword, 0, len(ifs)+len(fors))
	for _, ifblk := range ifs {
		out = append(out, ifToKeyword(ifblk))
	}
	for _, forblk := range fors {
		out = append(out, forToKeyword(forblk))
	}
	return out
}

func ifToKeyword(ifblk rdiff.If) rdiff.Keyword {
	kw := rdiff.Keyword{
		Name:   "IF",
		Type:   "IF",
		Status: ifblk.Status,
	}
	children := make([]rdiff.Keyword, 0, len(ifblk.Branches))
	for _, br := range ifblk.Branches {
		name := strings.TrimSpace(br.Type)
		if br.Condition != "" {
			name = name + " " + br.Condition
		}
		bkw := rdiff.Keyword{
			Name:   name,
			Type:   "BRANCH",
			Status: br.Status,
		}
		if len(br.Body) > 0 {
			bkw.Keywords = append(bkw.Keywords, orderedBodyToKeywords(br.Body)...)
		} else {
			bkw.Keywords = append(bkw.Keywords, br.Keywords...)
			bkw.Keywords = append(bkw.Keywords, controlFlowToKeywords(br.Ifs, br.Fors)...)
		}
		if br.Return != nil {
			bkw.Keywords = append(bkw.Keywords, returnToKeyword(*br.Return))
		}
		children = append(children, bkw)
	}
	kw.Keywords = children
	return kw
}

func forToKeyword(forblk rdiff.For) rdiff.Keyword {
	name := "FOR"
	if len(forblk.Var) > 0 {
		name = name + " " + strings.Join(forblk.Var, ", ")
	}
	if forblk.Flavor != "" {
		name = name + " " + forblk.Flavor
	}
	if len(forblk.Value) > 0 {
		name = name + " " + strings.Join(forblk.Value, ", ")
	}

	kw := rdiff.Keyword{
		Name:   name,
		Type:   "FOR",
		Status: forblk.Status,
	}

	children := make([]rdiff.Keyword, 0, len(forblk.Iter))
	for i, it := range forblk.Iter {
		iterKw := rdiff.Keyword{
			Name:   fmt.Sprintf("ITER %d", i+1),
			Type:   "ITER",
			Status: it.Status,
		}
		if len(it.Body) > 0 {
			iterKw.Keywords = append(iterKw.Keywords, orderedBodyToKeywords(it.Body)...)
		} else {
			iterKw.Keywords = append(iterKw.Keywords, it.Keywords...)
			iterKw.Keywords = append(iterKw.Keywords, controlFlowToKeywords(it.Ifs, it.Fors)...)
		}
		if it.Return != nil {
			iterKw.Keywords = append(iterKw.Keywords, returnToKeyword(*it.Return))
		}
		children = append(children, iterKw)
	}
	kw.Keywords = children
	return kw
}

func returnToKeyword(ret rdiff.Return) rdiff.Keyword {
	args := make([]string, 0, len(ret.Value))
	for _, v := range ret.Value {
		args = append(args, fmt.Sprintf("value=%s", v))
	}
	return rdiff.Keyword{
		Name:      "RETURN",
		Type:      "RETURN",
		Arguments: args,
		Status:    ret.Status,
	}
}

func buildKeywordsData(keywords []rdiff.Keyword) []map[string]any {
	result := make([]map[string]any, len(keywords))
	for i, kw := range keywords {
		children := keywordChildrenInOrder(kw)

		result[i] = map[string]any{
			"name":          kw.Name,
			"type":          kw.Type,
			"status":        kw.Status.Status,
			"statusMessage": strings.TrimSpace(kw.Status.Message),
			"start":         kw.Status.StartTime,
			"end":           kw.Status.EndTime,
			"arguments":     kw.Arguments,
			"keywords":      buildKeywordsData(children),
			"messages":      buildMessagesData(kw.Messages),
		}
	}
	return result
}

func keywordChildrenInOrder(kw rdiff.Keyword) []rdiff.Keyword {
	if len(kw.Body) > 0 {
		return orderedBodyToKeywords(kw.Body)
	}
	children := make([]rdiff.Keyword, 0, len(kw.Keywords)+len(kw.Ifs)+len(kw.Fors))
	children = append(children, kw.Keywords...)
	children = append(children, controlFlowToKeywords(kw.Ifs, kw.Fors)...)
	return children
}

func buildMessagesData(messages []rdiff.Message) []map[string]any {
	result := make([]map[string]any, len(messages))
	for i, msg := range messages {
		result[i] = map[string]any{
			"level":     msg.Level,
			"timestamp": msg.Timestamp,
			"html":      msg.HTML,
			"text":      msg.Text,
		}
	}
	return result
}

func buildSuitesData(suite *rdiff.Suite) []map[string]any {
	var result []map[string]any

	// Add current suite if it has tests
	if len(suite.Tests) > 0 {
		tests := make([]map[string]any, len(suite.Tests))
		for i, test := range suite.Tests {
			tests[i] = map[string]any{
				"name":   test.Name,
				"status": test.Status.Status,
			}
		}
		result = append(result, map[string]any{
			"name":  suite.Name,
			"tests": tests,
		})
	}

	// Recursively add sub-suites
	for i := range suite.Suites {
		result = append(result, buildSuitesData(&suite.Suites[i])...)
	}

	return result
}