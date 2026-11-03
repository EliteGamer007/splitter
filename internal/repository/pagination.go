package repository

func boundedPage(limit, offset, fallback int) (int, int) {
	if limit < 1 {
		limit = fallback
	}
	if limit > 50 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
