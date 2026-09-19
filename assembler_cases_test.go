package reasm_test

import (
	"testing"

	"reasm"
)

type reasmStep struct {
	ingest bool
	seq    uint32
	data   string
	want   string
}

func runReasmSteps(t *testing.T, limit int, steps []reasmStep) {
	t.Helper()
	a := reasm.New(limit)
	for i, s := range steps {
		if s.ingest {
			a.Ingest(s.seq, []byte(s.data))
			continue
		}
		got := string(a.Emit())
		if got != s.want {
			t.Fatalf("step %d Emit: got %q want %q", i, got, s.want)
		}
	}
}

func TestWraparound(t *testing.T) {
	cases := []struct {
		name  string
		steps []reasmStep
	}{
		{
			name: "跨过零界顺序到达",
			steps: []reasmStep{
				{ingest: true, seq: 0xFFFFFFF8, data: "01234567"},
				{ingest: true, seq: 0, data: "WXYZ"},
				{want: "01234567WXYZ"},
			},
		},
		{
			name: "回绕后乱序",
			steps: []reasmStep{
				{ingest: true, seq: 0, data: "WXYZ"},
				{ingest: true, seq: 0xFFFFFFF8, data: "01234567"},
				{want: "01234567WXYZ"},
			},
		},
		{
			name: "先发零界另一侧连续到达",
			steps: []reasmStep{
				{ingest: true, seq: 4, data: "WXYZ"},
				{ingest: true, seq: 0, data: "abcd"},
				{want: "abcdWXYZ"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runReasmSteps(t, 64, tc.steps) })
	}
}

func TestOverlapFirstWins(t *testing.T) {
	cases := []struct {
		name  string
		steps []reasmStep
	}{
		{
			name: "左侧重叠",
			steps: []reasmStep{
				{ingest: true, seq: 0, data: "abcd"},
				{ingest: true, seq: 2, data: "XY"},
				{want: "abcd"},
			},
		},
		{
			name: "右侧重叠只保留新字节",
			steps: []reasmStep{
				{ingest: true, seq: 4, data: "EFGH"},
				{ingest: true, seq: 0, data: "abCD"},
				{want: "abCDEFGH"},
			},
		},
		{
			name: "骑在已有数据左右两侧",
			steps: []reasmStep{
				{ingest: true, seq: 2, data: "CD"},
				{ingest: true, seq: 0, data: "abcd"},
				{want: "abCD"},
			},
		},
		{
			name: "回绕处左右重叠",
			steps: []reasmStep{
				{ingest: true, seq: 0xFFFFFFFE, data: "ab"},
				{ingest: true, seq: 2, data: "EF"},
				{ingest: true, seq: 0xFFFFFFFD, data: "Xabcdef"},
				{want: "XabcdEF"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runReasmSteps(t, 64, tc.steps) })
	}
}

func TestDuplicateDiscard(t *testing.T) {
	cases := []struct {
		name  string
		steps []reasmStep
	}{
		{
			name: "同段重发不改写",
			steps: []reasmStep{
				{ingest: true, seq: 0, data: "hello"},
				{ingest: true, seq: 0, data: "HELLO"},
				{want: "hello"},
			},
		},
		{
			name: "吐出后重发只吐一次",
			steps: []reasmStep{
				{ingest: true, seq: 0, data: "ab"},
				{want: "ab"},
				{ingest: true, seq: 0, data: "ab"},
				{want: ""},
			},
		},
		{
			name: "重发与新数据部分重叠",
			steps: []reasmStep{
				{ingest: true, seq: 0, data: "ab"},
				{ingest: true, seq: 0, data: "abXY"},
				{want: "abXY"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runReasmSteps(t, 64, tc.steps) })
	}
}

func TestHoleEmitsPrefixOnly(t *testing.T) {
	cases := []struct {
		name  string
		steps []reasmStep
	}{
		{
			name: "缺一截只吐前缀",
			steps: []reasmStep{
				{ingest: true, seq: 0, data: "ab"},
				{ingest: true, seq: 4, data: "cd"},
				{want: "ab"},
			},
		},
		{
			name: "空洞补齐后再吐后续",
			steps: []reasmStep{
				{ingest: true, seq: 0, data: "ab"},
				{ingest: true, seq: 4, data: "cd"},
				{want: "ab"},
				{ingest: true, seq: 2, data: "xy"},
				{want: "xycd"},
				{want: ""},
			},
		},
		{
			name: "连续等待直到补齐",
			steps: []reasmStep{
				{ingest: true, seq: 0, data: "ab"},
				{ingest: true, seq: 4, data: "cd"},
				{want: "ab"},
				{want: ""},
				{want: ""},
				{ingest: true, seq: 2, data: "xy"},
				{want: "xycd"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runReasmSteps(t, 64, tc.steps) })
	}
}

func TestLimitEvictsOldestHole(t *testing.T) {
	cases := []struct {
		name  string
		limit int
		steps []reasmStep
	}{
		{
			name:  "顶满时扔最老空洞之后的数据",
			limit: 8,
			steps: []reasmStep{
				{ingest: true, seq: 0, data: "abcd"},
				{ingest: true, seq: 6, data: "EFGH"},
				{ingest: true, seq: 12, data: "ij"},
				{want: "abcd"},
				{ingest: true, seq: 4, data: "ghEF"},
				{want: "ghEFGH"},
				{want: ""},
			},
		},
		{
			name:  "连续前缀占满时新片段只丢洞外部分",
			limit: 4,
			steps: []reasmStep{
				{ingest: true, seq: 0, data: "ab"},
				{ingest: true, seq: 4, data: "cd"},
				{ingest: true, seq: 2, data: "XYZW"},
				{want: "abXY"},
				{want: ""},
			},
		},
		{
			name:  "扔完老空洞后仍可补齐后续",
			limit: 8,
			steps: []reasmStep{
				{ingest: true, seq: 2, data: "XY"},
				{ingest: true, seq: 6, data: "GH"},
				{ingest: true, seq: 0, data: "ab"},
				{want: "abXY"},
				{ingest: true, seq: 4, data: "??"},
				{want: "??GH"},
			},
		},
		{
			name:  "上限为零不收数据",
			limit: 0,
			steps: []reasmStep{
				{ingest: true, seq: 0, data: "ab"},
				{want: ""},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runReasmSteps(t, tc.limit, tc.steps) })
	}
}
