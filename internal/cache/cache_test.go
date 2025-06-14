package cache_test

import (
	"testing"
	"time"

	"github.com/polyscone/tofu/internal/cache"
)

func TestCache(t *testing.T) {
	type Value struct {
		data int
	}

	c := cache.New[string, *Value]()

	v1 := c.LoadOrStore("foo", func() (*Value, time.Duration) { return &Value{data: 123}, 0 })
	v2 := c.LoadOrStore("foo", func() (*Value, time.Duration) { return &Value{data: 456}, 0 })

	if v1 != v2 {
		t.Error("want cached pointers to be the same")
	}
	if want, got := v1.data, v2.data; want != got {
		t.Errorf("want cached data to be %v; got %v", want, got)
	}
}
