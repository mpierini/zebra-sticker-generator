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

func TestEstimatedSizeCoversOutput(t *testing.T) {
	// A label with 6-digit dimensions makes every coordinate as long as the
	// overhead constants allow for.
	huge := Label{WidthDots: 999_999, HeightDots: 999_999, Font: "A", Pitch: 6, LineHeight: 10}
	cases := map[string]struct {
		lines []string
		label Label
	}{
		"default label":  {[]string{"  .:-+o$&#MN  ", "", strings.Repeat("N", 130)}, defaultLabel(t)},
		"one char rows":  {[]string{"N", "N", "N"}, huge},
		"6-digit coords": {[]string{strings.Repeat(".", 50), strings.Repeat("N", 50)}, huge},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			estimate, actual := estimatedSize(test.lines), len(Render(test.lines, test.label))
			if estimate < actual {
				t.Errorf("estimatedSize = %d, below actual output %d", estimate, actual)
			}
		})
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
