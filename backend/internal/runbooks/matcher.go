package runbooks

func Match(labels map[string]string, books []Definition) (Definition, bool) {
	for _, book := range books {
		if matches(book.Match, labels) {
			return book, true
		}
	}
	return Definition{}, false
}

func matches(selector, labels map[string]string) bool {
	for key, expected := range selector {
		if labels[key] != expected {
			return false
		}
	}
	return true
}
