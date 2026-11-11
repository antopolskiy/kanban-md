package web

import (
	"sort"
	"strconv"
)

func levelLinks(data *boardData) []navLink {
	if !data.tree.has && (data.query.Level == "" || data.query.Level == levelAll) {
		return nil
	}
	nums, unknown := levelChoices(data)
	q := data.query
	links := []navLink{{
		Href:   q.with(queryLevel, levelAll).href("/"),
		Label:  "All",
		Active: q.Level == "" || q.Level == levelAll,
	}}
	for _, n := range nums {
		val := strconv.Itoa(n)
		links = append(links, navLink{
			Href:   q.with(queryLevel, val).href("/"),
			Label:  "L" + val,
			Active: q.Level == val,
		})
	}
	if unknown {
		links = append(links, navLink{
			Href:   q.with(queryLevel, levelUnknown).href("/"),
			Label:  "L?",
			Active: q.Level == levelUnknown,
		})
	}
	return links
}

func levelChoices(data *boardData) ([]int, bool) {
	seen := map[int]bool{}
	var nums []int
	unknown := false
	for _, t := range data.active {
		depth := data.tree.depths[t.ID]
		if depth == unknownDepth {
			unknown = true
			continue
		}
		if seen[depth] {
			continue
		}
		seen[depth] = true
		nums = append(nums, depth)
	}
	sort.Ints(nums)
	return nums, unknown
}

func levelValueList(data *boardData) string {
	values := []string{levelAll}
	nums, unknown := levelChoices(data)
	for _, n := range nums {
		values = append(values, strconv.Itoa(n))
	}
	if unknown {
		values = append(values, levelUnknown)
	}
	return joinComma(values)
}

func joinComma(values []string) string {
	out := ""
	for i, value := range values {
		if i > 0 {
			out += ","
		}
		out += value
	}
	return out
}
