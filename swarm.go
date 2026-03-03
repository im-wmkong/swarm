package swarm

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
)

// Run 并发地对数据切片中的每个项执行 handle 函数。
func Run[T any](data []T, handle func(item T) error, opts ...Option) error {
	if len(data) == 0 {
		return nil
	}

	// 初始化默认配置
	options := &Options{
		concurrency: runtime.GOMAXPROCS(0), // 默认并发数为 CPU 核心数
		ctx:         context.Background(),
		failFast:    false,
		retryTimes:  0,
		panicHandler: func(p any) error {
			return fmt.Errorf("panic recovered: %v\nstack: %s", p, debug.Stack())
		},
	}

	// 应用用户传入的 Options
	for _, opt := range opts {
		opt(options)
	}

	// 准备执行环境
	// 如果开启了 FailFast，我们需要包装一个可取消的 context 来通知主循环停止派发任务
	ctx, cancel := context.WithCancel(options.ctx)
	defer cancel()

	errChan := make(chan error, len(data))
	sem := make(chan struct{}, options.concurrency)
	var wg sync.WaitGroup

	// 派发任务
loop:
	for _, item := range data {
		// 检查上下文是否已经被取消（外部取消或 FailFast 触发）
		select {
		case <-ctx.Done():
			errChan <- ctx.Err() // 记录 context 取消的错误
			break loop           // 停止派发新任务
		default:
		}

		// 获取并发令牌。这里也要结合 ctx，防止在等待令牌时死锁阻塞
		select {
		case <-ctx.Done():
			errChan <- ctx.Err()
			break loop
		case sem <- struct{}{}:
		}

		wg.Add(1)
		go func(val T) {
			defer wg.Done()
			defer func() { <-sem }() // 释放令牌

			// Panic 处理
			defer func() {
				if p := recover(); p != nil {
					if options.panicHandler != nil {
						if err := options.panicHandler(p); err != nil {
							errChan <- err
						}
					}
					// 发生了 Panic，如果开启了快速失败，直接取消后续任务
					if options.failFast {
						cancel()
					}
				}
			}()

			// 带有重试机制的执行逻辑
			var err error
			for i := 0; i <= options.retryTimes; i++ {
				// 每次执行前检查一下 context，如果已经取消了就没必要再重试或执行了
				if ctx.Err() != nil {
					err = ctx.Err()
					break
				}

				err = handle(val)
				if err == nil {
					break // 成功则跳出重试循环
				}
			}

			// 如果最终还是失败了，处理错误
			if err != nil {
				errChan <- err
				// 如果开启了快速失败，立刻通知主循环和其他刚启动的任务
				if options.failFast {
					cancel()
				}
			}
		}(item)
	}

	// 等待所有已派发的任务完成
	wg.Wait()
	close(errChan)

	// 聚合错误
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}
