package zpl

import (
	"strings"
	"testing"
)

func defaultLabel(t *testing.T) Label {
	t.Helper()
	label, err := NewLabel(4, 6, 203)
	if err != nil {
		t.Fatalf("NewLabel: %v", err)
	}
	return label
}

func TestNewLabelDefaults(t *testing.T) {
	label := defaultLabel(t)
	if label.WidthDots != 812 || label.HeightDots != 1218 {
		t.Errorf("size = %dx%d, want 812x1218", label.WidthDots, label.HeightDots)
	}
	if label.MaxCols() != 130 || label.MaxRows() != 118 {
		t.Errorf("grid = %dx%d, want 130x118", label.MaxCols(), label.MaxRows())
	}
}

func TestNewLabelTooSmall(t *testing.T) {
	cases := map[string]struct {
		widthInches, heightInches float64
		dpi                       int
	}{
		"narrower than margins": {0.1, 6, 203},
		"shorter than margins":  {4, 0.1, 203},
		"zero dpi":              {4, 6, 0},
		"negative size":         {-4, 6, 203},
	}
	for name, size := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := NewLabel(size.widthInches, size.heightInches, size.dpi); err == nil {
				t.Error("expected error")
			}
		})
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
	got := Render([]string{"a^b~c_d"}, defaultLabel(t))
	if !strings.Contains(got, "^FH_^FDa_5Eb_7Ec_5Fd^FS") {
		t.Errorf("control characters not escaped:\n%s", got)
	}
}

func TestRenderedSizeIsExact(t *testing.T) {
	label := defaultLabel(t)
	lines := []string{"  .:-+o$&#MN  ", "", "a^b~c_d", strings.Repeat("N", label.MaxCols())}
	cols := label.MaxCols()
	originX := max((label.WidthDots-cols*label.Pitch)/2, 0)
	originY := max((label.HeightDots-len(lines)*label.LineHeight)/2, 0)
	if got, want := renderedSize(lines, label, originX, originY), len(Render(lines, label)); got != want {
		t.Errorf("renderedSize = %d, want %d", got, want)
	}
}

func TestRenderAllocatesOnce(t *testing.T) {
	label := defaultLabel(t)
	lines := make([]string, label.MaxRows())
	for index := range lines {
		lines[index] = strings.Repeat(" .:-+o$&#MN", label.MaxCols()/11)
	}
	lines[3] = "escaped ^ ~ _ characters"
	allocs := testing.AllocsPerRun(20, func() { Render(lines, label) })
	if allocs != 1 {
		t.Errorf("Render made %.0f allocations, want 1", allocs)
	}
}
