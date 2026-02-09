package tui

const previewIdealWrap = 80

func wrapWidthForPreview(previewPaneWidth int) int {
	// Leave a bit of margin for styling/padding.
	w := previewPaneWidth - 2
	if w < 1 {
		w = 1
	}
	if w > previewIdealWrap {
		w = previewIdealWrap
	}
	return w
}
