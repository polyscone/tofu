package argon2_test

import (
	"bytes"
	"testing"

	"github.com/polyscone/tofu/internal/pkg/password/argon2"
	"github.com/polyscone/tofu/internal/pkg/size"
)

// Argon2 key derivation functions take a memory size parameter in kibibytes.
// So a memory parameter of 1024 actually means 1024 KiB, which is the
// same as 1 MiB.
// This constant just helps to keep the terminology straight in the tests,
// since it allows us to think in mebibytes.
// without having to convert the numbers in our head.
const mebibyte = 1 * size.Mebibyte / size.Kibibyte

func TestArgon2(t *testing.T) {
	tt := []struct {
		name        string
		variant     argon2.Variant
		password    string
		iterations  uint32
		memory      uint32
		parallelism uint8
		saltLength  uint32
		keyLength   uint32
		want        string
		wantError   bool
	}{
		{"argon2i,v=19,m=1024,t=1,p=1,salt=16,key=32", argon2.I, "correct horse battery staple", 1, mebibyte, 1, 16, 32, "$argon2i$v=19$m=1024,t=1,p=1$MDEyMzQ1Njc4OTAxMjM0NQ$bzciLtf1Rv5ofnr9dFF8JMyYKVChrt8BHfFNwstrYjw", false},
		{"argon2id,v=19,m=1024,t=32,p=4,salt=8,key=16", argon2.ID, "correct horse battery staple", 32, mebibyte, 4, 8, 16, "$argon2id$v=19$m=1024,t=32,p=4$MDEyMzQ1Njc$jowWQ4IVVI8pPkmoGo4tIg", false},
		{"argon2id,v=19,m=1024,t=32,p=4,salt=16,key=16", argon2.ID, "correct horse battery staple", 32, mebibyte, 4, 16, 16, "$argon2id$v=19$m=1024,t=32,p=4$MDEyMzQ1Njc4OTAxMjM0NQ$4WPzUYL3xMhH1onDrxLUKw", false},
		{"argon2id,v=19,m=1024,t=32,p=4,salt=16,key=32", argon2.ID, "correct horse battery staple", 32, mebibyte, 4, 16, 32, "$argon2id$v=19$m=1024,t=32,p=4$MDEyMzQ1Njc4OTAxMjM0NQ$UxK4e5uI4USZ1yGCdXqHG4+f1jEOXihdu2MYiCQPVBQ", false},
		{"argon2id,v=19,m=65536,t=3,p=2,salt=16,key=32", argon2.ID, "correct horse battery staple", 3, 64 * mebibyte, 2, 16, 32, "$argon2id$v=19$m=65536,t=3,p=2$MDEyMzQ1Njc4OTAxMjM0NQ$PgkJ+PZ2qE1sgfx/MIlJTisYA9AIBOobJiafadKsVg0", false},
		{"argon2id,v=19,m=65536,t=4,p=4,salt=16,key=32", argon2.ID, "correct horse battery staple", 4, 64 * mebibyte, 4, 16, 32, "$argon2id$v=19$m=65536,t=4,p=4$MDEyMzQ1Njc4OTAxMjM0NQ$phZlQ8JGKT6FUn5gZOfcKDOpng42WfJtVSkA7GBbPcw", false},

		{"error unknown variant", argon2.Variant("bad variant"), "error", 1, mebibyte, 1, 8, 16, "", true},

		{"argon2i error too few iterations", argon2.I, "error", 0, mebibyte, 1, 8, 16, "", true},
		{"argon2i error too little memory", argon2.I, "error", 1, 512, 1, 8, 16, "", true},
		{"argon2i error too little parallelism", argon2.I, "error", 1, mebibyte, 0, 8, 16, "", true},
		{"argon2i error salt too short", argon2.I, "error", 1, mebibyte, 1, 4, 16, "", true},
		{"argon2i error key too short", argon2.I, "error", 1, mebibyte, 1, 8, 8, "", true},

		{"argon2id error too few iterations", argon2.ID, "error", 0, mebibyte, 1, 8, 16, "", true},
		{"argon2id error too little memory", argon2.ID, "error", 1, 512, 1, 8, 16, "", true},
		{"argon2id error too little parallelism", argon2.ID, "error", 1, mebibyte, 0, 8, 16, "", true},
		{"argon2id error salt too short", argon2.ID, "error", 1, mebibyte, 1, 4, 16, "", true},
		{"argon2id error key too short", argon2.ID, "error", 1, mebibyte, 1, 8, 8, "", true},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			// All wanted strings are expecting the salt to be generated from
			// the series of bytes: "0123456789012345"
			// This is why the Argon2' Rand io.Reader is hard coded
			// as "0123456789012345"
			// If this hard coded value is changed then all encoded hashes
			// generated from the Argon2id function will be different and all
			// tests will break
			reader := bytes.NewReader([]byte("0123456789012345"))
			hash, err := argon2.EncodedHash(reader, []byte(tc.password), argon2.Params{
				Variant:     tc.variant,
				Iterations:  tc.iterations,
				Memory:      tc.memory,
				Parallelism: tc.parallelism,
				SaltLength:  tc.saltLength,
				KeyLength:   tc.keyLength,
			})

			if tc.wantError {
				if err == nil {
					t.Error("want error; got <nil>")
				}
			} else if err != nil {
				t.Error("want <nil>; got error")
			}

			if want, got := tc.want, string(hash); want != got {
				t.Errorf("want %q; got %q", want, got)
			}
		})
	}
}

func TestArgon2CSPRNG(t *testing.T) {
	tt := []struct {
		name    string
		variant argon2.Variant
		samples int
	}{
		{"argon2i no duplicates", argon2.I, 50},
		{"argon2id no duplicates", argon2.ID, 50},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			seen := make(map[string]struct{})
			for i := 0; i < tc.samples; i++ {
				hash, err := argon2.EncodedHash(nil, []byte("correct horse battery staple"), argon2.Params{
					Variant:     tc.variant,
					Iterations:  1,
					Memory:      mebibyte,
					Parallelism: 4,
					SaltLength:  8,
					KeyLength:   16,
				})
				if err != nil {
					t.Error("want <nil>; got error")
				}

				if _, ok := seen[string(hash)]; ok {
					t.Errorf("already seen %q; want unique encoded hash", hash)
				}

				seen[string(hash)] = struct{}{}
			}
		})
	}
}

func TestArgon2Verify(t *testing.T) {
	tt := []struct {
		name          string
		variant       argon2.Variant
		password      string
		encoded       string
		wantPassed    bool
		wantRehash    bool
		wantError     bool
		wantRehashErr bool
	}{
		{"argon2i pass", argon2.I, "correct horse battery staple", "$argon2i$v=19$m=1024,t=1,p=1$MDEyMzQ1Njc4OTAxMjM0NQ$bzciLtf1Rv5ofnr9dFF8JMyYKVChrt8BHfFNwstrYjw", true, false, false, false},
		{"argon2i fail", argon2.I, "correct horse battery", "$argon2i$v=19$m=1024,t=1,p=1$MDEyMzQ1Njc4OTAxMjM0NQ$bzciLtf1Rv5ofnr9dFF8JMyYKVChrt8BHfFNwstrYjw", false, false, false, false},
		{"argon2i version error", argon2.I, "correct horse battery staple", "$argon2i$v=20$m=1024,t=1,p=1$MDEyMzQ1Njc4OTAxMjM0NQ$bzciLtf1Rv5ofnr9dFF8JMyYKVChrt8BHfFNwstrYjw", false, false, true, false},
		{"argon2i needs rehash", argon2.I, "correct horse battery staple", "$argon2i$v=19$m=1024,t=1,p=1$MDEyMzQ1Njc4OTAxMjM0NQ$bzciLtf1Rv5ofnr9dFF8JMyYKVChrt8BHfFNwstrYjw", true, true, false, false},
		{"argon2i params error in rehash (bad variant in validate params)", argon2.I, "correct horse battery staple", "$argon2i$v=19$m=1024,t=1,p=1$MDEyMzQ1Njc4OTAxMjM0NQ$bzciLtf1Rv5ofnr9dFF8JMyYKVChrt8BHfFNwstrYjw", true, true, true, true},
		{"argon2i different salt", argon2.I, "correct horse battery staple", "$argon2i$v=19$m=1024,t=1,p=1$c29tZXNhbHRzb21lc2FsdA$1YTaVFHBui4l2zuN+PaYQ4zEXde9Hs5v3NgqT3Ycp+g", true, false, false, false},

		{"argon2id pass", argon2.ID, "correct horse battery staple", "$argon2id$v=19$m=1024,t=1,p=1$MDEyMzQ1Njc4OTAxMjM0NQ$dtZk5RFBYscPQNVuNtnrBFQ8X2yXI9sJEF6d+6qcX04", true, false, false, false},
		{"argon2id fail", argon2.ID, "correct horse battery", "$argon2id$v=19$m=1024,t=1,p=1$MDEyMzQ1Njc4OTAxMjM0NQ$dtZk5RFBYscPQNVuNtnrBFQ8X2yXI9sJEF6d+6qcX04", false, false, false, false},
		{"argon2id version error", argon2.ID, "correct horse battery staple", "$argon2id$v=20$m=1024,t=1,p=1$MDEyMzQ1Njc4OTAxMjM0NQ$dtZk5RFBYscPQNVuNtnrBFQ8X2yXI9sJEF6d+6qcX04", false, false, true, false},
		{"argon2id needs rehash", argon2.ID, "correct horse battery staple", "$argon2id$v=19$m=1024,t=1,p=1$MDEyMzQ1Njc4OTAxMjM0NQ$dtZk5RFBYscPQNVuNtnrBFQ8X2yXI9sJEF6d+6qcX04", true, true, false, false},
		{"argon2id params error in rehash (bad variant in validate params)", argon2.ID, "correct horse battery staple", "$argon2id$v=19$m=1024,t=1,p=1$MDEyMzQ1Njc4OTAxMjM0NQ$dtZk5RFBYscPQNVuNtnrBFQ8X2yXI9sJEF6d+6qcX04", true, true, true, true},
		{"argon2id different salt", argon2.ID, "correct horse battery staple", "$argon2id$v=19$m=1024,t=1,p=1$c29tZXNhbHRzb21lc2FsdA$RAME85Nb8CxpfIy5Ca0XN5R2weN/7wRhxgF0shgwz70", true, false, false, false},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			params := argon2.Params{
				Variant:     tc.variant,
				Iterations:  1,
				Memory:      mebibyte,
				Parallelism: 1,
				SaltLength:  16,
				KeyLength:   32,
			}
			if tc.wantRehash {
				if tc.wantRehashErr {
					params.Variant = argon2.Variant("bad variant")
				} else {
					if tc.variant == argon2.I {
						params.Variant = argon2.ID
					} else {
						params.Variant = argon2.I
					}
				}
				params.Iterations = 2
				params.Memory = 2 * mebibyte
				params.Parallelism = 2
				params.SaltLength = 32
				params.KeyLength = 64
			}

			passed, rehash, err := argon2.Verify([]byte(tc.password), []byte(tc.encoded), &params)

			if tc.wantError {
				if err == nil {
					t.Error("want error; got <nil>")
				}
			} else if err != nil {
				t.Error("want <nil>; got error")
			}

			if want, got := tc.wantPassed, passed; want != got {
				t.Errorf("want %v; got %v", want, got)
			}

			wantRehash := tc.wantRehash
			if tc.wantRehashErr {
				wantRehash = false
			}

			if want, got := wantRehash, rehash; want != got {
				t.Errorf("want %v; got %v", want, got)
			}
		})
	}
}
