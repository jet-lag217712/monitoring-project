package setup

func reviewListVisibleRows(height int, hasProgressRail bool) int {
	if height <= 0 {
		height = 24
	}
	overhead := 14
	if hasProgressRail {
		overhead = 16
	}
	visible := height - overhead
	if visible < 3 {
		return 3
	}
	return visible
}

func reviewScrollTopForCursor(cursor, scrollTop, total, visible int) int {
	if total <= 0 || visible <= 0 {
		return 0
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= total {
		cursor = total - 1
	}
	if cursor < scrollTop {
		return cursor
	}
	if cursor >= scrollTop+visible {
		return cursor - visible + 1
	}
	maxTop := total - visible
	if maxTop < 0 {
		return 0
	}
	if scrollTop > maxTop {
		return maxTop
	}
	return scrollTop
}
