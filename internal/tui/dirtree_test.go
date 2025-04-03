package tui

import (
	"fmt"
	"testing"
)

func TestDirTreeWalkDir(t *testing.T) {
	m := dirTree{}
	tv, err := m.walkDir("D:BSCS Spring 2022/7th Semester")
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range tv {
		fmt.Println(k, v)
	}
}

func BenchmarkDirTree_walkDir(b *testing.B) {
	m := dirTree{}
	for i := 0; i < b.N; i++ {
		m.walkDir("D:BSCS Spring 2022/7th Semester")
	}
}
