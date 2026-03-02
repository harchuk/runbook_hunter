package runbooks

import "sort"

func Match(labels map[string]string, books []Definition) (Definition, bool) {
	matched := make([]Definition, 0, len(books))
	for _, book := range books {
		if matches(book.Match, labels) {
			matched = append(matched, book)
		}
	}
	if len(matched) == 0 {
		return Definition{}, false
	}
	sort.SliceStable(matched, func(i, j int) bool {
		return len(matched[i].Match) > len(matched[j].Match)
	})
	return matched[0], true
}

func matches(selector, labels map[string]string) bool {
	for key, expected := range selector {
		if labels[key] != expected {
			return false
		}
	}
	return true
}
