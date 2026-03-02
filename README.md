# Swarm 🐝

`swarm` 是一个轻量级、类型安全的 Go 语言并发任务处理库。它提供了一种简单优雅的方式来并发处理切片数据，内部实现了可靠的并发度控制、Goroutine 防泄漏以及 Panic 安全恢复机制。

借助**函数式选项模式（Functional Options）**，`swarm` 提供了极高的扩展性，支持超时控制、快速失败（Fail-Fast）、任务重试等高级特性。

## ✨ 核心特性

* 🚀 **泛型支持**: 基于 Go 1.18+ 泛型编写，支持任意类型的数据切片。
* 🛡️ **Panic 安全**: 自动捕获并恢复单个任务中的 Panic，防止单个任务崩溃拖垮整个进程。
* ⚙️ **并发控制**: 允许自定义最大并发量（默认使用当前 CPU 核心数）。
* ⏱️ **Context 集成**: 原生支持 `context.Context`，轻松实现任务取消与超时控制。
* ⚡ **快速失败 (Fail-Fast)**: 开启后，任意一个任务失败即可立刻终止后续未开始的任务。
* 🔄 **重试机制**: 支持配置单个任务的失败重试次数，有效应对网络抖动等临时性故障。

## 📦 安装

使用 `go get` 将包引入到你的项目中：

```bash
go get github.com/im-wmkong/swarm
```

## 🚀 快速开始

### 基础用法

最简单的用法是传入数据切片和处理函数。`swarm` 会自动使用机器的 CPU 核心数作为并发量来执行任务，并在所有任务完成后聚合返回错误。

```go
package main

import (
	"fmt"
	"log"
	
	"github.com/im-wmkong/swarm"
)

func main() {
	data := []string{"apple", "banana", "cherry", "date"}

	// 并发处理每个项
	err := swarm.Run(data, func(item string) error {
		fmt.Printf("Processing: %s\n", item)
		return nil
	})

	if err != nil {
		log.Fatalf("Execution failed: %v", err)
	}
	fmt.Println("All done!")
}
```

### 高级用法 (使用 Options)

你可以通过组合不同的 `Option` 来应对复杂的业务场景。例如：限制并发数为 5、设置 3 秒超时、允许重试 2 次，并在遇到错误时立即停止（Fail-Fast）。

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/im-wmkong/swarm"
)

func main() {
	data := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	// 1. 设置 3 秒超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 2. 执行并发任务
	err := swarm.Run(data, func(item int) error {
		// 模拟网络请求或耗时操作
		time.Sleep(500 * time.Millisecond)
		
		if item == 7 {
			return errors.New("simulated network error")
		}
		
		fmt.Printf("Task %d completed\n", item)
		return nil
	}, 
		swarm.WithConcurrency(5),    // 限制最大并发数为 5
		swarm.WithContext(ctx),      // 传入超时 Context
		swarm.WithRetry(2),          // 失败后最多重试 2 次
		swarm.WithFailFast(),        // 如果重试后依然失败，立刻停止其他任务
	)

	if err != nil {
		fmt.Printf("Finished with errors:\n%v\n", err)
	}
}
```

## 🛠️ 配置选项 (Options API)

`swarm.Run` 支持以下可选配置：

| Option | 说明 | 默认值 |
| :--- | :--- | :--- |
| `WithConcurrency(n int)` | 设置最大并发执行的 Goroutine 数量。 | `runtime.GOMAXPROCS(0)` |
| `WithContext(ctx context.Context)` | 注入上下文，用于控制任务的取消或超时。 | `context.Background()` |
| `WithFailFast()` | 开启快速失败。任意任务报错后，不再调度新任务，并尝试中断进行中的重试。 | `false` |
| `WithRetry(times int)` | 单个任务执行失败时的重试次数（不含首次执行）。 | `0` (不重试) |
| `WithPanicHandler(fn)` | 自定义任务发生 Panic 时的处理回调。 | 记录堆栈并返回 `error` |

## 📄 许可证

本项目采用 [MIT License](LICENSE) 开源许可证。