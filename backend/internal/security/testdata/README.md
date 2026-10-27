`invalid-palette-ref.tiff` is copied from the `golang.org/x/image` v0.47.0 testdata.
The upstream project distributes it under its BSD 3-Clause license:

- Fixture: https://go.googlesource.com/image/+/refs/tags/v0.47.0/testdata/invalid-palette-ref.tiff
- License: https://go.googlesource.com/image/+/refs/tags/v0.47.0/LICENSE

It verifies that the TIFF decoder used by PocketBase's imaging path rejects an
out-of-range palette index before the vulnerable image-processing code can run.
