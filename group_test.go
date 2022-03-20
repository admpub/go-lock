package lock

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestGroup(t *testing.T) {
	mu := NewGroup(nil)
	mu.Lock("g")
	defer mu.Unlock("g")
	if mu.TryLock("g") {
		t.Errorf("cannot fetch mutex !!!")
	}
}

func TestGroupMutliWaitLock(t *testing.T) {
	var (
		wg sync.WaitGroup
		mu = NewGroup(nil)
		cn = 3
	)

	for i := 0; i < cn; i++ {
		wg.Add(1)
		go func() {
			mu.Lock("h")
			time.Sleep(1e7)
			mu.Unlock("h")
			wg.Done()
		}()
	}
	wg.Wait()

	for i := 0; i < cn; i++ {
		wg.Add(1)
		go func() {
			mu.Lock("g")
			time.Sleep(1e7)
			mu.UnlockAndFree("g")
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestGroupUnlockAndFree(t *testing.T) {
	var (
		wg sync.WaitGroup
		mu = NewGroup(nil)
		mg = mu.(*group)
	)

	for j := 1; j < 5; j++ {
		for i := 0; i < j; i++ {
			wg.Add(1)
			go func() {
				mu.Lock("h")
				time.Sleep(1e6)
				mu.UnlockAndFree("h")
				wg.Done()
			}()
		}
		wg.Wait()
		mg.mu.Lock()
		if _, ok := mg.group["h"]; ok {
			t.Error("h mutex exist after UnlockAndFree")
		}
		mg.mu.Unlock()
	}
}

func TestGroupTryLockFailedAndUnlockAndFree(t *testing.T) {
	var (
		wg sync.WaitGroup
		mu = NewGroup(nil)
		mg = mu.(*group)
	)

	for j := 1; j < 5; j++ {
		for i := 0; i < j; i++ {
			wg.Add(1)
			go func() {
				if mu.TryLock("h") {
					time.Sleep(1e6)
					mu.UnlockAndFree("h")
				}
				wg.Done()
			}()
		}
		wg.Wait()
		mg.mu.Lock()
		if _, ok := mg.group["h"]; ok {
			t.Error("h mutex exist after UnlockAndFree")
		}
		mg.mu.Unlock()
	}
}

func TestGroupTryLockTimeout(t *testing.T) {
	mu := NewGroup(nil)
	mu.Lock("g")
	go func() {
		time.Sleep(1 * time.Millisecond)
		mu.Unlock("g")
	}()
	if mu.TryLockWithTimeout("g", 500*time.Microsecond) {
		t.Errorf("cannot fetch mutex in 500us !!!")
	}
	if !mu.TryLockWithTimeout("g", 5*time.Millisecond) {
		t.Errorf("should fetch mutex in 5ms !!!")
	}
	mu.Unlock("g")
}

func TestGroupTryLockContext(t *testing.T) {
	mu := NewGroup(nil)
	ctx, cancel := context.WithCancel(context.Background())
	mu.Lock("g")
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	if mu.TryLockWithContext("g", ctx) {
		t.Errorf("cannot fetch mutex !!!")
	}
}

func BenchmarkGroupChanMutext(b *testing.B) {
	mu := NewGroup(nil)
	a := 0
	c := 0
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock("g")
			a++
			mu.UnlockAndFree("g")
			mu.Lock("g")
			c = a
			mu.UnlockAndFree("g")
		}
	})
	_ = a
	_ = c
}

func BenchmarkGroupCASMutex(b *testing.B) {
	mu := NewGroup(NewCASMutexInterface)
	a := 0
	c := 0
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock("g")
			a++
			mu.UnlockAndFree("g")
			mu.Lock("g")
			c = a
			mu.UnlockAndFree("g")
		}
	})
	_ = a
	_ = c
}
