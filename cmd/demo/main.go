package main

import (
	"fmt"

	"reasm"
)

func show(title string, limit int, parts [][2]any) {
	a := reasm.New(limit)
	for _, p := range parts {
		a.Ingest(p[0].(uint32), []byte(p[1].(string)))
	}
	fmt.Printf("%s => %q\n", title, string(a.Emit()))
}

func main() {
	fmt.Println("runtime demo")
	show("顺序", 64, [][2]any{{uint32(0), "ab"}, {uint32(2), "cd"}})
	show("轻微乱序", 64, [][2]any{{uint32(2), "cd"}, {uint32(0), "ab"}})
	show("回绕", 64, [][2]any{{uint32(0xFFFFFFF8), "01234567"}, {uint32(0), "WXYZ"}})
	show("左侧重叠", 64, [][2]any{{uint32(0), "abcd"}, {uint32(2), "XY"}})
	show("重复", 64, [][2]any{{uint32(0), "hello"}, {uint32(0), "HELLO"}})
	show("空洞", 64, [][2]any{{uint32(0), "ab"}, {uint32(4), "cd"}})
	show("超限", 8, [][2]any{{uint32(0), "abcd"}, {uint32(6), "EFGH"}, {uint32(12), "ijkl"}})
}
