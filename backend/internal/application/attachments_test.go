package application

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestAttachmentMediaValidatesExtensionAndContent(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"receipt.png", []byte("\x89PNG\r\n\x1a\nimage"), "image/png"},
		{"receipt.jpg", []byte{0xff, 0xd8, 0xff, 0, 1}, "image/jpeg"},
		{"receipt.pdf", []byte("%PDF-1.7 document"), "application/pdf"},
		{"fake.png", []byte("%PDF-1.7 document"), ""},
		{"empty.pdf", nil, ""},
		{"payload.exe", []byte("MZ executable"), ""},
		{"short.heic", []byte("0000ftypheic"), ""},
		{"short.heif", []byte("0000ftypmif1abcdefghijklmnopqrs"), ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := attachmentMedia(test.name, test.data)
			if test.want == "" && ok {
				t.Fatalf("attachmentMedia accepted invalid file as %q", got)
			}
			if test.want != "" && (!ok || got != test.want) {
				t.Fatalf("attachmentMedia = (%q, %v), want %q", got, ok, test.want)
			}
		})
	}
}

func officeArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range files {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = file.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestOfficeZipRejectsMacroEnabledDocuments(t *testing.T) {
	if !officeZipIsSafe(officeArchive(t, map[string]string{"word/document.xml": "<document/>", "[Content_Types].xml": "application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"}), ".docx") {
		t.Fatal("expected a non-macro Office document to be accepted")
	}
	for name, files := range map[string]map[string]string{
		"macro project":      {"word/document.xml": "<document/>", "word/vbaProject.bin": "macro"},
		"macro content type": {"[Content_Types].xml": "application/vnd.ms-excel.sheet.macroEnabled.main+xml"},
	} {
		t.Run(name, func(t *testing.T) {
			if officeZipIsSafe(officeArchive(t, files), ".docx") {
				t.Fatal("expected macro-enabled Office document to be rejected")
			}
		})
	}
}
