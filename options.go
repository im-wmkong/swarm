package swarm

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
)

// Option 定义了修改 options 的函数签名
type Option func(*options)

// options 定义了并发执行的各种配置
type options struct {
	concurrency  int
	ctx          context.Context
	failFast     bool
	panicHandler func(p any) error
	retryTimes   int
}

// defaultOptions 返回默认的 options 配置
func defaultOptions() *options {
	return &options{
		concurrency: runtime.GOMAXPROCS(0), // 默认并发数为 CPU 核心数
		ctx:         context.Background(),
		failFast:    false,
		retryTimes:  0,
		panicHandler: func(p any) error {
			return fmt.Errorf("panic recovered: %v\nstack: %s", p, debug.Stack())
		},
	}
}

// WithConcurrency 设置最大并发量。如果不设置，默认使用当前机器的 CPU 核心数。
func WithConcurrency(n int) Option {
	return func(o *options) {
		if n > 0 {
			o.concurrency = n
		}
	}
}

// WithContext 注入上下文，可用于超时控制或外部主动取消任务。
func WithContext(ctx context.Context) Option {
	return func(o *options) {
		if ctx != nil {
			o.ctx = ctx
		}
	}
}

// WithFailFast 开启快速失败模式。任意一个任务失败，将不再调度后续未开始的任务。
func WithFailFast() Option {
	return func(o *options) {
		o.failFast = true
	}
}

// WithPanicHandler 自定义 Panic 捕获和处理逻辑。
// 如果传入 nil，则忽略此选项，继续使用默认的处理逻辑（记录堆栈并返回 error）。
// 如果想要完全吞噬或忽略 Panic 错误，请传入一个返回 nil 的函数：func(p any) error { return nil }
func WithPanicHandler(handler func(p any) error) Option {
	return func(o *options) {
		if handler != nil {
			o.panicHandler = handler
		}
	}
}

// WithRetry 设置单个任务失败时的重试次数（不包含首次执行）。
func WithRetry(times int) Option {
	return func(o *options) {
		if times > 0 {
			o.retryTimes = times
		}
	}
}
