package swarm

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// 测试基础执行：所有任务正常完成
func TestRun_Basic(t *testing.T) {
	data := []int{1, 2, 3, 4, 5}
	var count int32

	err := Run(data, func(item int) error {
		atomic.AddInt32(&count, 1)
		return nil
	}, WithConcurrency(2))

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if int(count) != len(data) {
		t.Fatalf("expected %d tasks to run, got %d", len(data), count)
	}
}

// 测试快速失败 (Fail-Fast)：遇到错误立即停止派发新任务
func TestRun_FailFast(t *testing.T) {
	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	var executedCount int32

	err := Run(data, func(item int) error {
		atomic.AddInt32(&executedCount, 1)
		// 模拟任务耗时，确保有时间取消
		time.Sleep(10 * time.Millisecond)

		if item == 3 {
			return errors.New("mock error on item 3")
		}
		return nil
	}, WithConcurrency(1), WithFailFast()) // 并发设为 1，确保顺序执行以稳定测试

	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	// 因为并发是 1，且执行到第 3 个报错触发 Fail-Fast，
	// 所以执行总数应该远小于 10（理想情况下是 3）。
	if int(executedCount) == len(data) {
		t.Fatalf("FailFast did not work, all %d tasks were executed", len(data))
	}
	t.Logf("Fail-Fast triggered. Executed tasks: %d", executedCount)
}

// 测试重试机制：前几次失败，最后一次成功
func TestRun_Retry(t *testing.T) {
	data := []int{1}
	var attempts int32

	err := Run(data, func(item int) error {
		currentAttempt := atomic.AddInt32(&attempts, 1)
		if currentAttempt <= 2 {
			return errors.New("temporary error")
		}
		return nil
	}, WithRetry(2)) // 允许重试 2 次（总共最多执行 3 次）

	if err != nil {
		t.Fatalf("expected task to succeed after retries, but got error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

// 测试上下文超时控制
func TestRun_ContextTimeout(t *testing.T) {
	data := []int{1, 2, 3, 4, 5}
	var executedCount int32

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := Run(data, func(item int) error {
		// 每个任务耗时 30ms
		time.Sleep(30 * time.Millisecond)
		atomic.AddInt32(&executedCount, 1)
		return nil
	}, WithConcurrency(1), WithContext(ctx))

	if err == nil {
		t.Fatal("expected context deadline exceeded error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got: %v", err)
	}

	// 50ms 的超时，并发为 1，每个任务 30ms，最多只能完成 1 个任务
	if int(executedCount) == len(data) {
		t.Fatal("expected execution to be halted by context timeout")
	}
	t.Logf("Context timeout triggered. Executed tasks: %d", executedCount)
}

// 测试 Panic 捕获与恢复
func TestRun_PanicRecovery(t *testing.T) {
	data := []int{1}

	err := Run(data, func(item int) error {
		panic("something went terribly wrong")
	})

	if err == nil {
		t.Fatal("expected an error due to panic, got nil")
	}

	if !strings.Contains(err.Error(), "panic recovered") {
		t.Fatalf("expected error to contain panic info, got: %v", err)
	}
}
