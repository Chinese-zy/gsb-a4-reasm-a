package reasm

import "sort"

type frag struct {
	seq  uint32
	data []byte
	ord  int
}

// Assembler 按序号把分片拼回字节流。
// Ingest 是送入，Emit 是取出，这两处调用方式需要保持不变。
type Assembler struct {
	limit int
	frags []frag
	ord   int
}

func New(limit int) *Assembler {
	if limit < 0 {
		limit = 0
	}
	return &Assembler{limit: limit}
}

func (a *Assembler) Ingest(seq uint32, payload []byte) {
	if len(payload) == 0 {
		return
	}
	buf := make([]byte, len(payload))
	copy(buf, payload)
	a.ord++
	a.frags = append(a.frags, frag{seq: seq, data: buf, ord: a.ord})
}

func (a *Assembler) Emit() []byte {
	if len(a.frags) == 0 {
		return nil
	}
	frags := append([]frag(nil), a.frags...)
	sort.Slice(frags, func(i, j int) bool {
		if frags[i].seq == frags[j].seq {
			return frags[i].ord < frags[j].ord
		}
		return frags[i].seq < frags[j].seq
	})

	start := frags[0].seq
	end := start
	for _, f := range frags {
		e := f.seq + uint32(len(f.data))
		if e >= start && e > end {
			end = e
		}
	}
	size := int(end - start)
	if size < 0 || size > 1<<20 {
		size = 0
	}

	buf := make([]byte, size)
	filled := make([]bool, size)
	var extra []byte
	for _, f := range frags {
		off := int(int64(f.seq) - int64(start))
		if off < 0 || off > len(buf) {
			extra = append(extra, f.data...)
			continue
		}
		for i := 0; i < len(f.data); i++ {
			at := off + i
			if at >= len(buf) {
				extra = append(extra, f.data[i:]...)
				break
			}
			buf[at] = f.data[i]
			filled[at] = true
		}
	}

	out := make([]byte, 0, len(buf)+len(extra))
	for i := 0; i < len(buf); i++ {
		if filled[i] {
			out = append(out, buf[i])
		}
	}
	out = append(out, extra...)
	a.frags = nil
	return out
}
