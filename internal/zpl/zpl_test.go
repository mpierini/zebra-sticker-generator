package zpl

import (
	"strings"
	"testing"
)

func TestNewLabelDefaults(t *testing.T) {
	label := NewLabel(4, 6, 203)
	if label.WidthDots != 812 || label.HeightDots != 1218 {
		t.Errorf("size = %dx%d, want 812x1218", label.WidthDots, label.HeightDots)
	}
	if label.MaxCols() != 130 || label.MaxRows() != 118 {
		t.Errorf("grid = %dx%d, want 130x118", label.MaxCols(), label.MaxRows())
	}
}

func TestRender(t *testing.T) {
	label := Label{WidthDots: 100, HeightDots: 50, Font: "A", Pitch: 6, LineHeight: 10}
	got := Render([]string{"@@", "  ", ".."}, label)
	want := "^XA\n^PW100\n^LL50\n^CI28\n" +
		"^FO44,10^AAN^FD@@^FS\n" +
		"^FO44,30^AAN^FD..^FS\n" +
		"^XZ\n"
	if got != want {
		t.Errorf("Render =\n%s\nwant\n%s", got, want)
	}
}

func TestRenderEscapesControlCharacters(t *testing.T) {
	got := Render([]string{"a^b~c_d"}, NewLabel(4, 6, 203))
	if !strings.Contains(got, "^FH_^FDa_5Eb_7Ec_5Fd^FS") {
		t.Errorf("control characters not escaped:\n%s", got)
	}
}
