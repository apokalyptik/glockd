package main

import (
	"strconv"
	"testing"

	"github.com/apokalyptik/glockc"
)

func BenchmarkInspectNothing(b *testing.B) {
	c1, _ := glockc.New("127.0.0.1", 9999)
	for i := 0; i < b.N; i++ {
		c1.Inspect("BenchmarkInspectNothing", false)
	}
}

func BenchmarkInspect(b *testing.B) {
	c1, _ := glockc.New("127.0.0.1", 9999)
	c2, _ := glockc.New("127.0.0.1", 9999)
	c1.Get("BenchmarkInspect", false)
	for i := 0; i < b.N; i++ {
		c2.Inspect("BenchmarkInspect", false)
	}
}

func BenchmarkGet(b *testing.B) {
	c1, _ := glockc.New("127.0.0.1", 9999)
	for i := 0; i < b.N; i++ {
		c1.Get("BenchmarkGet"+strconv.Itoa(i), false)
	}
}

func BenchmarkReGet(b *testing.B) {
	c1, _ := glockc.New("127.0.0.1", 9999)
	for i := 0; i < b.N; i++ {
		c1.Get("BenchmarkReGet", false)
	}
}

func BenchmarkGotRelease(b *testing.B) {
	c1, _ := glockc.New("127.0.0.1", 9999)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		c1.Get("BenchmarkGotRelease", false)
		b.StartTimer()
		c1.Release("BenchmarkGotRelease", false)
	}
}

func BenchmarkReleaseNothing(b *testing.B) {
	c1, _ := glockc.New("127.0.0.1", 9999)
	for i := 0; i < b.N; i++ {
		c1.Release("BenchmarkReleaseNothing", false)
	}
}

func BenchmarkGetFailure(b *testing.B) {
	c1, _ := glockc.New("127.0.0.1", 9999)
	c2, _ := glockc.New("127.0.0.1", 9999)
	c1.Get("BenchmarkGetFailure", false)
	for i := 0; i < b.N; i++ {
		c2.Get("BenchmarkGetFailure", false)
	}
}

func BenchmarkSharedInspectNothing(b *testing.B) {
	c1, _ := glockc.New("127.0.0.1", 9999)
	for i := 0; i < b.N; i++ {
		c1.Inspect("BenchmarkSharedInspectNothing", false)
	}
}

func BenchmarkSharedInspect(b *testing.B) {
	c1, _ := glockc.New("127.0.0.1", 9999)
	c2, _ := glockc.New("127.0.0.1", 9999)
	c1.Get("BenchmarkSharedInspect", false)
	for i := 0; i < b.N; i++ {
		c2.Inspect("BenchmarkSharedInspect", false)
	}
}

func BenchmarkSharedGet(b *testing.B) {
	c1, _ := glockc.New("127.0.0.1", 9999)
	for i := 0; i < b.N; i++ {
		c1.Get("BenchmarkSharedGet"+strconv.Itoa(i), false)
	}
}

func BenchmarkSharedReGet(b *testing.B) {
	c1, _ := glockc.New("127.0.0.1", 9999)
	for i := 0; i < b.N; i++ {
		c1.Get("BenchmarkSharedReGet", false)
	}
}

func BenchmarkSharedGotRelease(b *testing.B) {
	c1, _ := glockc.New("127.0.0.1", 9999)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		c1.Get("BenchmarkSharedGotRelease", false)
		b.StartTimer()
		c1.Release("BenchmarkSharedGotRelease", false)
	}
}

func BenchmarkSharedReleaseNothing(b *testing.B) {
	c1, _ := glockc.New("127.0.0.1", 9999)
	for i := 0; i < b.N; i++ {
		c1.Release("BenchmarkSharedReleaseNothing", false)
	}
}

func TestExclusiveLocks(t *testing.T) {
	var c1 glockc.Client
	var c2 glockc.Client
	var err error
	var r int

	// Connect Clients
	c1, err = glockc.New("127.0.0.1", 9999)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	c2, err = glockc.New("127.0.0.1", 9999)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}

	// foo should be unlocked
	r, err = c1.Inspect("foo", false)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if r != 0 {
		t.Errorf("expected foo to be unlocked")
	}

	// we should get a lock on foo
	r, err = c1.Get("foo", false)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if r != 1 {
		t.Errorf("expected to aquire foo")
	}

	// foo should be locked
	r, err = c2.Inspect("foo", false)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if r != 1 {
		t.Errorf("expected foo to look locked")
	}

	// client 2 should not be able to aquire a lock on foo since client1 has it
	r, err = c2.Get("foo", false)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if r == 1 {
		t.Errorf("expected to be denied a lock on foo")
	}

	// client 1 should be able to reaquire their lock on foo
	r, err = c1.Get("foo", false)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if r != 1 {
		t.Errorf("expected to have foo")
	}

	// release lock
	r, err = c1.Release("foo", false)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if r != 1 {
		t.Errorf("expected to have released foo")
	}

	// client 2 should now be able to aquire a lock on foo
	r, err = c2.Get("foo", false)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if r != 1 {
		t.Errorf("expected to aquire foo")
	}

}

func TestSharedLocks(t *testing.T) {
	var c1 glockc.Client
	var c2 glockc.Client
	var c3 glockc.Client
	var err error
	var r int

	// Connect Clients
	c1, err = glockc.New("127.0.0.1", 9999)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	c2, err = glockc.New("127.0.0.1", 9999)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	c3, err = glockc.New("127.0.0.1", 9999)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}

	// foo should be unlocked
	r, err = c1.Inspect("foo", true)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if r != 0 {
		t.Errorf("expected foo to be unlocked")
	}

	// we should get a lock on foo
	r, err = c1.Get("foo", true)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if r != 1 {
		t.Errorf("expected to aquire foo")
	}

	// foo should be locked
	r, err = c2.Inspect("foo", true)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if r != 1 {
		t.Errorf("expected foo to look locked")
	}

	// client 2 should  be able to aquire a lock on foo
	r, err = c2.Get("foo", true)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if r != 2 {
		t.Errorf("expected to be second locker on foo, got %d", r)
	}

	// foo should be locked twice
	r, err = c3.Inspect("foo", true)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if r != 2 {
		t.Errorf("expected foo to be locked twice, got %d", r)
	}

	// client 1 should be able to reaquire their lock on foo
	r, err = c1.Get("foo", true)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if r != 2 {
		t.Errorf("expected to have foo, two locks")
	}

	// release lock
	r, err = c1.Release("foo", true)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if r != 1 {
		t.Errorf("expected to have released foo")
	}

	// foo should be locked once
	r, err = c3.Inspect("foo", true)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if r != 1 {
		t.Errorf("expected foo to be locked once, got %d", r)
	}

	// release lock
	r, err = c2.Release("foo", true)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if r != 1 {
		t.Errorf("expected to have released foo")
	}

	// foo should be unlocked
	r, err = c3.Inspect("foo", true)
	if err != nil {
		t.Errorf("expected no error, got %s", err)
	}
	if r != 0 {
		t.Errorf("expected foo to be unlocked")
	}
}
