package model

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/paginator"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// 定义样式
var (
	baseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("63"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			MarginLeft(2)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true).
			MarginLeft(2)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginLeft(2)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("35")).
			Bold(true).
			MarginLeft(2)
)

// KeyMap 定义键盘映射
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Help     key.Binding
	Quit     key.Binding
	Sort     key.Binding
	Filter   key.Binding
	Export   key.Binding
	Reset    key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Home     key.Binding
	End      key.Binding
}

// ShortHelp 返回简短帮助信息
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp 返回完整帮助信息
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.PageUp, k.PageDown, k.Home, k.End},
		{k.Sort, k.Filter, k.Reset, k.Export},
		{k.Help, k.Quit},
	}
}

// 默认键盘映射
var Keys = KeyMap{
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "上移"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "下移"),
	),
	Left: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←", "左移"),
	),
	Right: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "右移"),
	),
	Help: key.NewBinding(
		key.WithKeys("h"),
		key.WithHelp("h", "帮助"),
	),
	Quit: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "退出"),
	),
	Sort: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "排序当前列"),
	),
	Filter: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "筛选当前列"),
	),
	Export: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "导出CSV"),
	),
	Reset: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "重置筛选"),
	),
	PageUp: key.NewBinding(
		key.WithKeys("shift+up"),
		key.WithHelp("Shift+↑", "上翻页（行）"),
	),
	PageDown: key.NewBinding(
		key.WithKeys("shift+down"),
		key.WithHelp("Shift+↓", "下翻页（行）"),
	),
	Home: key.NewBinding(
		key.WithKeys("shift+left"),
		key.WithHelp("Shift+←", "首列"),
	),
	End: key.NewBinding(
		key.WithKeys("shift+right"),
		key.WithHelp("Shift+→", "末列"),
	),
}

// TableModel 表格模型
type TableModel struct {
	Table         table.Model
	Title         string
	RowCount      int
	QueryDuration time.Duration
	Help          help.Model
	Keys          KeyMap
	Paginator     paginator.Model
	ShowHelp      bool
	SortColumn    int             // 当前排序的列
	SortAsc       bool            // 是否升序排序
	OriginalRows  []table.Row     // 保存原始数据行
	FilteredRows  []table.Row     // 保存过滤后的数据行
	AllRows       []table.Row     // 保存所有原始数据行（用于恢复）
	FilterText    string          // 过滤文本
	Filtering     bool            // 是否在过滤状态
	StatusMsg     string          // 状态消息
	TableColumns  []table.Column  // 表格列定义
	ScrollOffset  int             // 横向滚动偏移量
	MaxColumns    int             // 当前能显示的最大列数
	Width         int             // 当前表格宽度
	Height        int             // 当前表格高度
	TextInput     textinput.Model // 文本输入模型，用于筛选
}

// NewTableModel 初始化表格模型
func NewTableModel() TableModel {
	h := help.New()
	h.ShowAll = true

	p := paginator.New()
	p.Type = paginator.Dots
	p.PerPage = 1
	p.Page = 0
	p.SetTotalPages(1)

	// 初始化文本输入模型
	ti := textinput.New()
	ti.Placeholder = "输入筛选文本..."
	ti.Focus()
	ti.Width = 20
	ti.CharLimit = 50

	return TableModel{
		Help:         h,
		Keys:         Keys,
		Paginator:    p,
		ShowHelp:     false,
		SortColumn:   -1, // 默认不排序
		SortAsc:      true,
		Filtering:    false,
		ScrollOffset: 0, // 初始无偏移
		TextInput:    ti,
	}
}

// ExportToCSV 导出表格数据到CSV文件
func (m *TableModel) ExportToCSV() error {
	// 创建输出目录
	if err := os.MkdirAll("output", 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 打开文件用于写入
	outputFile := "output/查询结果.csv"
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %v", err)
	}
	defer f.Close()

	// 写入表头
	var headers []string
	for _, col := range m.TableColumns {
		headers = append(headers, col.Title)
	}
	fmt.Fprintln(f, strings.Join(headers, ","))

	// 写入数据行
	for _, row := range m.OriginalRows {
		var values []string
		for _, cell := range row {
			// 处理CSV特殊字符，如逗号、引号等
			cellStr := fmt.Sprintf("%v", cell)
			if strings.Contains(cellStr, ",") || strings.Contains(cellStr, "\"") || strings.Contains(cellStr, "\n") {
				cellStr = "\"" + strings.ReplaceAll(cellStr, "\"", "\"\"") + "\""
			}
			values = append(values, cellStr)
		}
		fmt.Fprintln(f, strings.Join(values, ","))
	}

	return nil
}

// GetDefaultTableStyles 返回默认的表格样式
func GetDefaultTableStyles() table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		Bold(true).
		Foreground(lipgloss.Color("87"))

	s.Selected = s.Selected.
		Foreground(lipgloss.Color("231")).
		Background(lipgloss.Color("63")).
		Bold(true)

	s.Cell = s.Cell.
		PaddingLeft(1).
		PaddingRight(1)

	return s
}

// Init 实现 tea.Model 接口
func (m TableModel) Init() tea.Cmd {
	return nil
}

// Update 实现 tea.Model 接口
func (m TableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// 如果正在过滤状态，使用textinput处理输入
		if m.Filtering {
			switch msg.String() {
			case "enter":
				// 确认过滤
				m.Filtering = false
				// 应用过滤
				m.FilterText = m.TextInput.Value()
				m.ApplyFilter()
				m.TextInput.Reset()
			case "esc":
				// 取消过滤
				m.Filtering = false
				m.FilterText = ""
				m.TextInput.Reset()
				// 恢复原始数据
				if len(m.AllRows) > 0 {
					m.OriginalRows = make([]table.Row, len(m.AllRows))
					copy(m.OriginalRows, m.AllRows)
				}
				m.UpdateVisibleColumns()
			default:
				// 使用textinput模型处理输入
				var inputCmd tea.Cmd
				m.TextInput, inputCmd = m.TextInput.Update(msg)
				return m, inputCmd
			}
			return m, nil
		}

		// 非过滤状态下的键盘操作
		switch {
		case key.Matches(msg, m.Keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.Keys.Help):
			m.ShowHelp = !m.ShowHelp
		case key.Matches(msg, m.Keys.Sort):
			// 排序功能
			// 当前显示的列的索引就是ScrollOffset
			targetColumn := m.ScrollOffset
			m.SortColumn = targetColumn
			m.SortAsc = !m.SortAsc
			m.SortRows()

			// 确保光标可见
			m.EnsureCursorVisible()
		case key.Matches(msg, m.Keys.Filter):
			// 开始过滤
			// 当前显示的列的索引就是ScrollOffset
			targetColumn := m.ScrollOffset
			m.Filtering = true
			m.FilterText = ""
			m.TextInput.Reset()
			m.TextInput.Focus()
			if len(m.AllRows) == 0 && len(m.OriginalRows) > 0 {
				m.AllRows = make([]table.Row, len(m.OriginalRows))
				copy(m.AllRows, m.OriginalRows)
			}
			m.SortColumn = targetColumn
			return m, textinput.Blink
		case key.Matches(msg, m.Keys.Reset):
			if len(m.AllRows) > 0 {
				m.OriginalRows = make([]table.Row, len(m.AllRows))
				copy(m.OriginalRows, m.AllRows)
				m.FilterText = ""
				m.StatusMsg = "已重置筛选器，恢复全部数据"
				m.UpdateVisibleColumns()
			}
		case key.Matches(msg, m.Keys.Export):
			err := m.ExportToCSV()
			if err != nil {
				m.StatusMsg = fmt.Sprintf("导出失败: %v", err)
			} else {
				m.StatusMsg = "导出成功: ./output/查询结果.csv"
			}
		case key.Matches(msg, m.Keys.Left):
			if m.ScrollOffset > 0 {
				currentCursor := m.Table.Cursor() // 保存光标
				m.ScrollOffset--
				m.UpdateVisibleColumns()
				m.Table.SetCursor(currentCursor) // 恢复光标
				m.EnsureCursorVisible()          // 确保可见
			}
		case key.Matches(msg, m.Keys.Right):
			// 检查是否还可以向右滚动
			if m.ScrollOffset < len(m.TableColumns)-1 {
				currentCursor := m.Table.Cursor() // 保存光标
				m.ScrollOffset++
				m.UpdateVisibleColumns()
				m.Table.SetCursor(currentCursor) // 恢复光标
				m.EnsureCursorVisible()          // 确保可见
			}
		case key.Matches(msg, m.Keys.End):
			// 计算最大滚动偏移，确保最后一列可见
			maxScroll := len(m.TableColumns) - 1
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.ScrollOffset != maxScroll {
				// 保存当前光标位置
				currentCursor := m.Table.Cursor()
				m.ScrollOffset = maxScroll
				m.UpdateVisibleColumns()
				// 恢复光标位置并确保可见
				m.Table.SetCursor(currentCursor)
				m.EnsureCursorVisible()
			}
		case key.Matches(msg, m.Keys.Home):
			if m.ScrollOffset > 0 {
				// 保存当前光标位置
				currentCursor := m.Table.Cursor()
				m.ScrollOffset = 0
				m.UpdateVisibleColumns()
				// 恢复光标位置并确保可见
				m.Table.SetCursor(currentCursor)
				m.EnsureCursorVisible()
			}
		case key.Matches(msg, m.Keys.PageUp):
			pageSize := m.Height / 2

			// 通过多次发送up键实现翻页
			for i := 0; i < pageSize; i++ {
				upMsg := tea.KeyMsg{Type: tea.KeyUp}
				newTable, _ := m.Table.Update(upMsg)
				m.Table = newTable
			}

		case key.Matches(msg, m.Keys.PageDown):
			pageSize := m.Height / 2

			// 通过多次发送down键实现翻页
			for i := 0; i < pageSize; i++ {
				downMsg := tea.KeyMsg{Type: tea.KeyDown}
				newTable, _ := m.Table.Update(downMsg)
				m.Table = newTable
			}
		}
	case tea.WindowSizeMsg:
		h, v := baseStyle.GetFrameSize()
		m.Width = msg.Width - h
		// 预留空间给标题、信息行和帮助文本
		reservedHeight := 4 // 标题、信息行、状态消息、帮助文本等
		if m.Filtering {
			reservedHeight++
		}
		if m.FilterText != "" {
			reservedHeight++
		}
		if m.StatusMsg != "" {
			reservedHeight++
		}
		m.Height = msg.Height - v - reservedHeight
		m.Table.SetWidth(m.Width)
		m.Table.SetHeight(m.Height)
		m.CalculateMaxColumns()
		m.UpdateVisibleColumns()
	}

	newTable, newCmd := m.Table.Update(msg)
	m.Table = newTable
	if newCmd != nil {
		cmd = newCmd
	}

	m.EnsureCursorVisible()

	return m, cmd
}

// View 实现 tea.Model 接口
func (m TableModel) View() string {
	var b strings.Builder

	// 标题行和查询时间
	queryDuration := "0s"
	if m.QueryDuration > 0 {
		queryDuration = m.QueryDuration.String()
	}
	titleInfo := fmt.Sprintf("%s | 查询耗时: %v", m.Title, queryDuration)
	b.WriteString(titleStyle.Render(titleInfo))
	b.WriteString("\n")

	// 计算列信息
	maxScroll := len(m.TableColumns) - m.MaxColumns
	if maxScroll < 0 {
		maxScroll = 0
	}

	// 合并行列和排序信息到一行
	currentRow := m.Table.Cursor() + 1
	var navigationInfo string

	if m.Filtering {
		// 在筛选状态下显示筛选信息而不是导航信息
		var columnName string
		if m.SortColumn < len(m.TableColumns) {
			columnName = m.TableColumns[m.SortColumn].Title
		} else {
			columnName = fmt.Sprintf("第%d列", m.SortColumn+1)
		}

		filterInfo := fmt.Sprintf("筛选中 (列: %s): ", columnName)
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			MarginLeft(2).
			Render(filterInfo))

		inputStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Background(lipgloss.Color("0")).
			Bold(true)

		b.WriteString(inputStyle.Render(m.TextInput.View()))
	} else {
		// 非筛选状态下显示常规导航信息和筛选结果
		if m.SortColumn >= 0 {
			var columnName string
			if m.SortColumn < len(m.TableColumns) {
				columnName = m.TableColumns[m.SortColumn].Title
			} else {
				columnName = fmt.Sprintf("第%d列", m.SortColumn+1)
			}
			navigationInfo = fmt.Sprintf("行: %d/%d (%.1f%%) | 列: %d-%d/%d | 排序: %s (%s)",
				currentRow, m.RowCount, float64(currentRow)*100/float64(m.RowCount),
				m.ScrollOffset+1, m.ScrollOffset+m.MaxColumns, len(m.TableColumns),
				columnName,
				map[bool]string{true: "升序", false: "降序"}[m.SortAsc])
		} else {
			navigationInfo = fmt.Sprintf("行: %d/%d (%.1f%%) | 列: %d-%d/%d",
				currentRow, m.RowCount, float64(currentRow)*100/float64(m.RowCount),
				m.ScrollOffset+1, m.ScrollOffset+m.MaxColumns, len(m.TableColumns))
		}

		// 如果有筛选结果，添加到导航信息后面
		if m.FilterText != "" {
			var columnName string
			if m.SortColumn < len(m.TableColumns) {
				columnName = m.TableColumns[m.SortColumn].Title
			} else {
				columnName = fmt.Sprintf("第%d列", m.SortColumn+1)
			}
			filterInfo := fmt.Sprintf(" | 筛选: %s=\"%s\" (%d行)",
				columnName, m.FilterText, len(m.OriginalRows))
			navigationInfo += filterInfo
		}

		b.WriteString(infoStyle.Render(navigationInfo))
	}
	b.WriteString("\n")

	// 状态消息（只显示非筛选相关的状态消息）
	if m.StatusMsg != "" && !strings.Contains(m.StatusMsg, "筛选") && !strings.Contains(m.StatusMsg, "恢复全部数据") {
		b.WriteString(statusStyle.Render(m.StatusMsg))
		b.WriteString("\n")
	}

	// 表格内容
	b.WriteString(baseStyle.Render(m.Table.View()))

	// 计算剩余空间并添加填充
	_, height, _ := term.GetSize(int(os.Stdout.Fd()))
	currentHeight := strings.Count(b.String(), "\n") // the help text and margins
	if height > currentHeight {
		padding := height - currentHeight - 1 // -1 to leave room for help text
		b.WriteString(strings.Repeat("\n", padding))
	}

	// 帮助信息
	if m.ShowHelp {
		// 将所有帮助信息合并为一行
		var helpBindings []string
		for _, bindings := range m.Keys.FullHelp() {
			for _, binding := range bindings {
				keys := binding.Help().Key
				desc := binding.Help().Desc
				if keys != "" && desc != "" {
					helpBindings = append(helpBindings, fmt.Sprintf("%s %s", keys, desc))
				}
			}
		}
		helpText := strings.Join(helpBindings, " | ")
		b.WriteString(helpStyle.Render(helpText))
	} else {
		helpText := "按 h 显示帮助 | Shift+←/→ 首/末列 | Shift+↑/↓ 行翻页 | s 排序当前列 | f 筛选当前列 | r 重置 | e 导出 | esc 退出"
		b.WriteString(helpStyle.Render(helpText))
	}

	return b.String()
}

// CalculateMaxColumns 计算当前可以显示的最大列数
func (m *TableModel) CalculateMaxColumns() {
	if len(m.TableColumns) == 0 {
		m.MaxColumns = 0
		return
	}

	totalWidth := 0
	count := 0

	for i, col := range m.TableColumns {
		if i < m.ScrollOffset {
			continue
		}

		totalWidth += col.Width + 2
		if totalWidth > m.Width {
			break
		}
		count++
	}

	if count == 0 && len(m.TableColumns) > m.ScrollOffset {
		count = 1
	}

	m.MaxColumns = count
}

// UpdateVisibleColumns 更新可见列
func (m *TableModel) UpdateVisibleColumns() {
	if len(m.TableColumns) == 0 {
		return
	}

	// 确保ScrollOffset不会超出范围
	if m.ScrollOffset < 0 {
		m.ScrollOffset = 0
	}
	if m.ScrollOffset >= len(m.TableColumns) {
		m.ScrollOffset = len(m.TableColumns) - 1
	}

	m.CalculateMaxColumns()

	// 调整ScrollOffset以确保最后几列可见
	if m.ScrollOffset > 0 && m.ScrollOffset+m.MaxColumns > len(m.TableColumns) {
		// 如果当前滚动位置会导致右边出现空白，往左调整
		newOffset := len(m.TableColumns) - m.MaxColumns
		if newOffset < 0 {
			newOffset = 0
		}
		m.ScrollOffset = newOffset
	}

	endIdx := m.ScrollOffset + m.MaxColumns
	if endIdx > len(m.TableColumns) {
		endIdx = len(m.TableColumns)
	}

	// 保存当前光标位置、焦点状态和表格尺寸
	currentCursor := m.Table.Cursor()
	isFocused := m.Table.Focused()
	tableWidth := m.Width
	tableHeight := m.Height

	visibleColumns := m.TableColumns[m.ScrollOffset:endIdx]

	visibleRows := make([]table.Row, len(m.OriginalRows))
	for i, row := range m.OriginalRows {
		if i < len(visibleRows) && m.ScrollOffset < len(row) {
			visibleRow := make(table.Row, len(visibleColumns))
			for j := range visibleColumns {
				colIdx := m.ScrollOffset + j
				if colIdx < len(row) {
					visibleRow[j] = row[colIdx]
				} else {
					visibleRow[j] = ""
				}
			}
			visibleRows[i] = visibleRow
		}
	}

	newTable := table.New(
		table.WithColumns(visibleColumns),
		table.WithRows(visibleRows),
		table.WithFocused(isFocused),
		table.WithHeight(tableHeight),
	)

	// 确保表格尺寸保持一致
	newTable.SetWidth(tableWidth)
	newTable.SetHeight(tableHeight)

	newTable.SetStyles(GetDefaultTableStyles())

	// 设置光标位置并确保可见
	if len(visibleRows) > 0 {
		validCursor := currentCursor
		if validCursor < 0 {
			validCursor = 0
		} else if validCursor >= len(visibleRows) {
			validCursor = len(visibleRows) - 1
		}
		newTable.SetCursor(validCursor)
	}

	m.Table = newTable

	// 使用更直接的方法确保光标刷新和可见
	// 如果有行数，根据光标位置选择不同的刷新策略
	if len(m.Table.Rows()) > 0 {
		currentCursor := m.Table.Cursor()

		if currentCursor == 0 {
			// 如果光标在第一行，先下移再上移
			downMsg := tea.KeyMsg{Type: tea.KeyDown}
			newTable, _ := m.Table.Update(downMsg)
			m.Table = newTable

			upMsg := tea.KeyMsg{Type: tea.KeyUp}
			newTable, _ = m.Table.Update(upMsg)
			m.Table = newTable
		} else if currentCursor == len(m.Table.Rows())-1 {
			// 如果光标在最后一行，先上移再下移
			upMsg := tea.KeyMsg{Type: tea.KeyUp}
			newTable, _ := m.Table.Update(upMsg)
			m.Table = newTable

			downMsg := tea.KeyMsg{Type: tea.KeyDown}
			newTable, _ = m.Table.Update(downMsg)
			m.Table = newTable
		} else {
			// 如果光标在中间行，可以使用任意顺序
			upMsg := tea.KeyMsg{Type: tea.KeyUp}
			newTable, _ := m.Table.Update(upMsg)
			m.Table = newTable

			downMsg := tea.KeyMsg{Type: tea.KeyDown}
			newTable, _ = m.Table.Update(downMsg)
			m.Table = newTable
		}
	}
}

// EnsureCursorVisible 确保光标在可见范围内
func (m *TableModel) EnsureCursorVisible() {
	cursor := m.Table.Cursor()
	rowCount := len(m.Table.Rows())

	// 如果表格为空，不做任何操作
	if rowCount == 0 {
		return
	}

	// 如果光标超出范围，重置到有效范围内
	if cursor < 0 {
		m.Table.SetCursor(0)
		return
	} else if cursor >= rowCount {
		m.Table.SetCursor(rowCount - 1)
		return
	}

	// 即使在范围内，也检查是否真的有选中行
	selectedRow := m.Table.SelectedRow()
	if len(selectedRow) == 0 {
		// 如果没有选中行（光标看不见），强制重新设置光标位置
		// 首先尝试当前位置
		m.Table.SetCursor(cursor)

		// 如果仍然没有选中行，尝试移到第一行
		selectedRow = m.Table.SelectedRow()
		if len(selectedRow) == 0 && rowCount > 0 {
			m.Table.SetCursor(0)
		}
	}
}

// SortRows 排序表格行
func (m *TableModel) SortRows() {
	if m.SortColumn < 0 {
		return
	}

	rows := make([]table.Row, len(m.OriginalRows))
	copy(rows, m.OriginalRows)

	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i][m.SortColumn], rows[j][m.SortColumn]
		aNum, aErr := strconv.ParseFloat(a, 64)
		bNum, bErr := strconv.ParseFloat(b, 64)

		if aErr == nil && bErr == nil {
			if m.SortAsc {
				return aNum < bNum
			}
			return aNum > bNum
		}

		if m.SortAsc {
			return a < b
		}
		return a > b
	})

	m.OriginalRows = rows
	m.UpdateVisibleColumns()
}

// ApplyFilter 应用过滤
func (m *TableModel) ApplyFilter() {
	if m.FilterText == "" {
		if len(m.AllRows) > 0 {
			m.OriginalRows = make([]table.Row, len(m.AllRows))
			copy(m.OriginalRows, m.AllRows)
		}
		m.UpdateVisibleColumns()
		m.StatusMsg = "已恢复全部数据"
		return
	}

	var filteredRows []table.Row
	lowerFilterText := strings.ToLower(m.FilterText)

	for _, row := range m.AllRows {
		cellValue := strings.ToLower(row[m.SortColumn])
		if strings.Contains(cellValue, lowerFilterText) {
			filteredRows = append(filteredRows, row)
		}
	}

	m.FilteredRows = filteredRows
	m.OriginalRows = make([]table.Row, len(filteredRows))
	copy(m.OriginalRows, filteredRows)

	m.UpdateVisibleColumns()

	columnName := m.TableColumns[m.SortColumn].Title
	m.StatusMsg = fmt.Sprintf("筛选结果: 在列 [%s] 中找到 %d 行匹配数据",
		columnName, len(filteredRows))
}

// TableData 表格数据结构
type TableData struct {
	Title    string            // 表格标题
	Headers  []string          // 表头
	Rows     [][]string        // 数据行
	Metadata map[string]string // 元数据（可选）
}

// ShowTable 显示表格数据
func ShowTable(data TableData) error {
	// 创建表格列
	tableColumns := make([]table.Column, len(data.Headers))
	colMaxWidth := make([]int, len(data.Headers))

	// 初始化列宽度
	for i, header := range data.Headers {
		colMaxWidth[i] = len(header) + 2
	}

	// 计算每列的最大宽度
	for _, row := range data.Rows {
		for i, cell := range row {
			if i < len(colMaxWidth) && len(cell) > colMaxWidth[i] {
				colMaxWidth[i] = len(cell) + 2
			}
		}
	}

	// 设置列属性
	for i, header := range data.Headers {
		width := colMaxWidth[i]
		if width < 10 {
			width = 10
		} else if width > 40 {
			width = 40
		}
		tableColumns[i] = table.Column{
			Title: header,
			Width: width,
		}
	}

	// 转换数据行格式
	tableRows := make([]table.Row, len(data.Rows))
	for i, row := range data.Rows {
		tableRows[i] = table.Row(row)
	}

	// 如果没有数据，添加一个空行
	if len(tableRows) == 0 {
		emptyRow := make(table.Row, len(data.Headers))
		for i := range emptyRow {
			emptyRow[i] = "无数据"
		}
		tableRows = append(tableRows, emptyRow)
	}

	// 创建表格模型
	m := NewTableModel()
	m.Title = data.Title
	m.RowCount = len(data.Rows)
	m.OriginalRows = make([]table.Row, len(tableRows))
	copy(m.OriginalRows, tableRows)
	m.AllRows = make([]table.Row, len(tableRows))
	copy(m.AllRows, tableRows)
	m.TableColumns = tableColumns

	// 设置表格尺寸
	width, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err == nil {
		m.Width = width - 4
		m.Height = height - 12
		if m.Height < 10 {
			m.Height = 10
		}
	} else {
		m.Width = 80
		m.Height = 20
	}

	// 初始化表格
	t := table.New(
		table.WithColumns(tableColumns),
		table.WithRows(tableRows),
		table.WithFocused(true),
		table.WithHeight(m.Height),
	)

	durationStr := data.Metadata["QueryDuration"]
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		// 处理错误
		return err
	}
	m.QueryDuration = duration
	// 设置表格样式
	t.SetStyles(GetDefaultTableStyles())
	m.Table = t

	// 计算可见列数并更新视图
	m.CalculateMaxColumns()
	m.UpdateVisibleColumns()

	// 运行程序
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// 其他方法（如Init、Update、View）可以根据需要添加
