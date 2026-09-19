package reasm_test

import (
	"testing"

	"reasm"
)

func TestContiguousAndSlightReorder(t *testing.T) {
	cases := []struct {
		name string
		in   []struct {
			seq  uint32
			data string
		}
		want string
	}{
		{
			name: "顺序",
			in: []struct {
				seq  uint32
				data string
			}{{0, "ab"}, {2, "cd"}},
			want: "abcd",
		},
		{
			name: "轻微乱序",
			in: []struct {
				seq  uint32
				data string
			}{{2, "cd"}, {0, "ab"}},
			want: "abcd",
		},
		{
			name: "三段颠倒",
			in: []struct {
				seq  uint32
				data string
			}{{4, "ef"}, {0, "ab"}, {2, "cd"}},
			want: "abcdef",
		},
		{
			name: "不从零开始",
			in: []struct {
				seq  uint32
				data string
			}{{100, "foo"}, {103, "bar"}},
			want: "foobar",
		},
		{
			name: "空片段忽略",
			in: []struct {
				seq  uint32
				data string
			}{{0, ""}, {0, "ok"}},
			want: "ok",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := reasm.New(64)
			for _, p := range tc.in {
				a.Ingest(p.seq, []byte(p.data))
			}
			if got := string(a.Emit()); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestEmitBetweenContiguousChunks(t *testing.T) {
	a := reasm.New(64)
	a.Ingest(0, []byte("ab"))
	if got := string(a.Emit()); got != "ab" {
		t.Fatalf("first %q", got)
	}
	a.Ingest(2, []byte("cd"))
	if got := string(a.Emit()); got != "cd" {
		t.Fatalf("second %q", got)
	}
	if got := a.Emit(); len(got) != 0 {
		t.Fatalf("third %q", got)
	}
}
