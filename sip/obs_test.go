package sip

import (
	"fmt"
	"testing"
)

func TestObserver(t *testing.T) {
	s := NewObserver()
	go func() {
		s.Notify("1")
		s.Notify("2")
	}()
	var i int
	s.data.Range(func(key, value any) bool {
		i++
		return true
	})
	fmt.Println("i:", i)
}
