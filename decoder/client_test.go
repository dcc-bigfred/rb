package decoder

import "testing"

func TestParseSoundListing(t *testing.T) {
	html := `
<td><input placeholder='F11_Decouple.wav'> </td><td>file</td><td align='right'>108</td>
<td><input placeholder='idle.wav'> </td><td>file</td><td align='right'>12</td>
`
	got := parseSoundListing([]byte(html))
	if len(got) != 2 {
		t.Fatalf("got %d files, want 2: %+v", len(got), got)
	}
	if got[0].Name != "F11_Decouple.wav" || got[0].SizeKB != 108 {
		t.Fatalf("file 0 = %+v", got[0])
	}
	if got[1].Name != "idle.wav" || got[1].SizeKB != 12 {
		t.Fatalf("file 1 = %+v", got[1])
	}
}
