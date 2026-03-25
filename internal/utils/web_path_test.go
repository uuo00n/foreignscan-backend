package utils

import "testing"

func TestNormalizeToUploadsWebPath(t *testing.T) {
	uploadDir := "/tmp/foreignscan-uploads"
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "relative uploads path", input: "uploads/images/r1/p1/a.jpg", want: "/uploads/images/r1/p1/a.jpg"},
		{name: "leading slash uploads path", input: "/uploads/labels/r1/predict/a.jpg", want: "/uploads/labels/r1/predict/a.jpg"},
		{name: "relative labels path", input: "labels/r1/predict/a.jpg", want: "/uploads/labels/r1/predict/a.jpg"},
		{name: "windows slashes", input: `uploads\\labels\\r1\\predict\\a.jpg`, want: "/uploads/labels/r1/predict/a.jpg"},
		{name: "absolute in upload dir", input: "/tmp/foreignscan-uploads/labels/r1/predict/a.jpg", want: "/uploads/labels/r1/predict/a.jpg"},
		{name: "absolute outside upload dir", input: "/tmp/other/a.jpg", want: ""},
		{name: "http url not allowed", input: "http://example.com/a.jpg", want: ""},
		{name: "empty", input: "", want: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeToUploadsWebPath(tc.input, uploadDir)
			if got != tc.want {
				t.Fatalf("normalizeToUploadsWebPath(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestNormalizeToStoredUploadsPath(t *testing.T) {
	got := NormalizeToStoredUploadsPath("/uploads/labels/r1/predict/a.jpg")
	want := "uploads/labels/r1/predict/a.jpg"
	if got != want {
		t.Fatalf("NormalizeToStoredUploadsPath got %q, want %q", got, want)
	}
}
