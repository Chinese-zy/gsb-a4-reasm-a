package reasm

// Assembler 按 32 位回绕序号把分片拼回字节流。
// Ingest 是送入，Emit 是取出，这两处调用方式保持不变。
//
// 规则：
//   - 序号按无符号 32 位回绕比较，最近半个序号空间内的分片视为新数据。
//   - 重叠位置以先确认（先送入）的字节为准，后到的重复字节丢弃。
//   - 存在空洞时 Emit 只吐出从头开始的连续前缀，空洞之后的内容继续缓存。
//   - 缓存字节数受 limit 限制；顶满时丢弃最老空洞之后（序号最小）的
//     乱序缓存，腾出空间给新到的连续前缀。
type Assembler struct {
	limit   int
	origin  int64 // 序号展开基准，固定为首个分片的绝对序号
	base    int64 // 下一段待取出的连续字节对应的绝对序号
	prefix  int64 // 连续前缀末端（开区间），buf 始终填满 [base, prefix)
	emitted bool
	has     bool
	buf     map[int64]byte
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
	if !a.has {
		a.origin = int64(seq)
		a.base = a.origin
		a.prefix = a.origin
		a.buf = make(map[int64]byte)
		a.has = true
	}

	i0 := unwrap(a.origin, seq)
	if a.emitted && i0+int64(len(payload)) <= a.base {
		// 整段都已经吐出或更老，按重复数据丢弃。
		return
	}

	for i, b := range payload {
		at := i0 + int64(i)
		if at < a.base && a.emitted {
			continue
		}
		if _, ok := a.buf[at]; ok {
			// 已确认的字节优先，重复内容丢弃。
			continue
		}
		for len(a.buf) >= a.limit {
			if !a.evictOldestIsland() {
				// 连续前缀占满额度时，空洞之外的新字节无法收留。
				return
			}
		}
		a.buf[at] = b
		if at == a.prefix {
			for {
				if _, ok := a.buf[a.prefix]; !ok {
					break
				}
				a.prefix++
			}
		}
	}

	// 首次吐出之前，更早到齐的分片把连续起点向前延伸。
	if !a.emitted {
		for {
			if _, ok := a.buf[a.base-1]; !ok {
				break
			}
			a.base--
		}
	}
}

// evictOldestIsland 丢弃连续前缀之后序号最小的缓存字节。
// 连续前缀 [base, prefix) 本身不允许丢弃。
func (a *Assembler) evictOldestIsland() bool {
	var oldest int64
	found := false
	for at := range a.buf {
		if at < a.prefix {
			continue
		}
		if !found || at < oldest {
			oldest = at
			found = true
		}
	}
	if !found {
		return false
	}
	delete(a.buf, oldest)
	return true
}

func (a *Assembler) Emit() []byte {
	if !a.has || a.prefix == a.base {
		return nil
	}
	n := int(a.prefix - a.base)
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = a.buf[a.base+int64(i)]
	}
	for at := a.base; at < a.prefix; at++ {
		delete(a.buf, at)
	}
	a.base = a.prefix
	a.emitted = true
	return out
}

// unwrap 把 32 位序号 seq 展开成以 origin 为基准的绝对序号。
func unwrap(origin int64, seq uint32) int64 {
	ref := uint32(origin)
	delta := seq - ref
	var rel int64
	if delta <= 1<<31 {
		rel = int64(delta)
	} else {
		rel = int64(delta) - 1<<32
	}
	return origin + rel
}
