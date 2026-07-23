package media

import "testing"

func TestExtForContentType(t *testing.T) {
	cases := map[string]string{
		"image/jpeg": ".jpg",
		"image/jpg":  ".jpg",
		"image/png":  ".png",
		"image/webp": ".webp",
		"IMAGE/PNG":  ".png",
	}
	for ct, want := range cases {
		got, err := ExtForContentType(ct)
		if err != nil {
			t.Fatalf("%s: %v", ct, err)
		}
		if got != want {
			t.Fatalf("%s: got %q want %q", ct, got, want)
		}
	}
	if _, err := ExtForContentType("application/pdf"); err != ErrInvalidType {
		t.Fatalf("expected ErrInvalidType, got %v", err)
	}
}

func TestFolders(t *testing.T) {
	if got := FolderBrand("b1"); got != "/scaloo/brands/b1" {
		t.Fatalf("FolderBrand: %q", got)
	}
	if got := FolderAvatar("u1"); got != "/scaloo/avatars/u1" {
		t.Fatalf("FolderAvatar: %q", got)
	}
}

func TestNewFileName(t *testing.T) {
	name, err := NewFileName("abc", "image/png")
	if err != nil || name != "abc.png" {
		t.Fatalf("got %q %v", name, err)
	}
}
