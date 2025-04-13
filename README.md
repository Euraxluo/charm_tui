# charm_tui

一个基于 Go 语言开发的美观、易用的终端用户界面（TUI）库，使用 [charm](https://github.com/charmbracelet) 工具包构建。

## 特点

- 🌈 美观的终端表格显示
- 🔄 支持表格列排序
- 🔍 支持表格内容筛选
- ⌨️ 键盘快捷键操作
- 📏 自适应终端大小
- 📊 横向滚动，支持大数据表格
- 📁 CSV 导出功能

## 安装

```bash
go get github.com/euraxluo/charm_tui
```

## 使用方法

以下是一个简单的例子，展示如何使用 charm_tui 显示表格数据：

```go
package main

import (
    "log"
    
    "github.com/euraxluo/charm_tui/model"
)

func main() {
    // 创建表格数据
    data := model.TableData{
        Title:   "示例表格",
        Headers: []string{"ID", "姓名", "年龄", "职业", "城市", "薪资"},
        Rows: [][]string{
            {"1", "张三", "28", "工程师", "北京", "15000"},
            {"2", "李四", "32", "设计师", "上海", "12000"},
            // ... 更多数据
        },
        Metadata: map[string]string{
            "QueryDuration": "20ms", // 可选的元数据
        },
    }

    // 显示表格
    if err := model.ShowTable(data); err != nil {
        log.Fatalf("显示表格失败: %v", err)
    }
}
```

## 键盘快捷键

| 快捷键 | 功能 |
|-------|------|
| `↑` / `↓` | 上下移动光标 |
| `←` / `→` | 左右移动（滚动列） |
| `Shift+↑` / `Shift+↓` | 上/下翻页 |
| `Shift+←` / `Shift+→` | 跳转到首列/末列 |
| `s` | 对当前列排序 |
| `f` | 筛选当前列 |
| `r` | 重置筛选 |
| `e` | 导出为 CSV |
| `h` | 显示/隐藏帮助 |
| `Esc` | 退出 |

## 功能

- **排序功能**: 按 `s` 键对当前选中列进行排序，再次按下切换升序/降序
- **筛选功能**: 按 `f` 键进入筛选模式，输入要筛选的文本并按回车
- **导出功能**: 按 `e` 键将当前表格内容导出为 CSV 文件
- **分页浏览**: 使用 `Shift+↑` 和 `Shift+↓` 进行快速翻页

## 示例

查看 [example](./example) 目录获取更多使用示例。

## 贡献

欢迎提交 Pull Request 或提出 Issue。

## 许可证

MIT 许可证 - 详见 [LICENSE](./LICENSE) 文件。
