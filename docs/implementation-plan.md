# MiniClaw Go 功能扩展实现计划

> 创建日期：2026-02-10
> 目标：实现 Skills 系统、MCP 接入、定时任务三大功能

---

## 目录

1. [概述](#1-概述)
2. [Skills 系统实现计划](#2-skills-系统实现计划)
3. [MCP 接入实现计划](#3-mcp-接入实现计划)
4. [定时任务实现计划](#4-定时任务实现计划)
5. [实施路线图](#5-实施路线图)
6. [测试策略](#6-测试策略)

---

## 1. 概述

### 1.1 目标

为 MiniClaw Go 添加以下三个核心功能，使其功能对齐 Java 版本：

1. **Skills 系统**：支持技能文件定义、解析、选择和热更新
2. **MCP 接入**：实现 Model Context Protocol 支持，允许外部工具通过标准协议注册
3. **定时任务**：支持 Cron 表达式定义的定时任务调度

### 1.2 现有架构分析

**已有基础**：
- 消息总线（`bus.InMemoryMessageBus`）
- 工具系统（`tools.ToolRegistry`、`tools.ToolExecutor`）
- Agent 服务（`agent.Agent`）
- 存储服务（`storage.FileSystemStorage`）
- 配置管理（`config.FileConfigManager`）

**可复用组件**：
- 工具注册表：Skills 可以转换为工具注册
- 消息总线：用于事件通知和热更新
- 文件系统存储：用于存储技能文件和任务配置

---

## 2. Skills 系统实现计划

### 2.1 功能概述

Skills 系统允许用户通过 Markdown 文件定义可重用的技能，Agent 可以根据上下文自动选择和激活合适的技能。

### 2.2 核心组件

| 组件 | 文件路径 | 职责 |
|------|----------|------|
| `Skill` | `internal/skills/skill.go` | 技能数据结构 |
| `SkillParser` | `internal/skills/parser.go` | 解析技能文件 |
| `SkillRegistry` | `internal/skills/registry.go` | 技能注册表 |
| `SkillSelector` | `internal/skills/selector.go` | 技能选择器 |
| `SkillFileWatcher` | `internal/skills/watcher.go` | 文件监控和热更新 |
| `SkillIndexBuilder` | `internal/skills/index.go` | 技能索引构建器 |
| `SkillGatingService` | `internal/skills/gating.go` | 技能权限控制 |

### 2.3 技能文件格式

```markdown
---
name: "code_review"
description: "Review code for best practices and potential issues"
category: "development"
tags: ["code", "review", "quality"]
requires: ["read_file"]
---

# Code Review Skill

This skill helps you review code for best practices, security issues, and performance optimizations.

## When to Use

Use this skill when:
- User asks for code review
- User mentions "review" or "check my code"
- User shares code snippets

## How to Use

1. Read the code file using `read_file`
2. Analyze the code for:
   - Security vulnerabilities
   - Performance issues
   - Code style violations
   - Best practices adherence
3. Provide actionable feedback

## Example

User: "Can you review this file?"
Assistant: I'll review the code for you.
```

### 2.4 实现步骤

#### Phase 1：基础数据结构（1-2天）

**文件**：`internal/skills/skill.go`

```go
type Skill struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Description string            `json:"description"`
    Category    string            `json:"category"`
    Tags        []string          `json:"tags"`
    Requires    []string          `json:"requires"`
    Content     string            `json:"content"`
    Metadata    map[string]string `json:"metadata"`
    Enabled     bool              `json:"enabled"`
    CreatedAt   time.Time         `json:"created_at"`
    UpdatedAt   time.Time         `json:"updated_at"`
}

type SkillTrigger struct {
    Keywords   []string `json:"keywords"`
    Intent     string   `json:"intent"`
    Confidence float64  `json:"confidence"`
}
```

#### Phase 2：技能解析器（2-3天）

**文件**：`internal/skills/parser.go`

```go
type SkillParser struct {
    storage storage.Storage
}

func (p *SkillParser) Parse(ctx context.Context, path string) (*Skill, error) {
    content, err := p.storage.ReadFile(ctx, path)
    if err != nil {
        return nil, err
    }

    return p.ParseContent(string(content), path)
}

func (p *SkillParser) ParseContent(content, path string) (*Skill, error) {
    parts := strings.SplitN(content, "---", 3)
    if len(parts) < 3 {
        return nil, fmt.Errorf("invalid skill format")
    }

    frontMatter := parts[1]
    skillContent := parts[2]

    var metadata map[string]interface{}
    if err := yaml.Unmarshal([]byte(frontMatter), &metadata); err != nil {
        return nil, err
    }

    skill := &Skill{
        ID:          generateSkillID(path),
        Name:        getString(metadata, "name"),
        Description: getString(metadata, "description"),
        Category:    getString(metadata, "category"),
        Tags:        getStringSlice(metadata, "tags"),
        Requires:    getStringSlice(metadata, "requires"),
        Content:     skillContent,
        Metadata:    extractMetadata(metadata),
        Enabled:     getBool(metadata, "enabled", true),
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }

    return skill, nil
}
```

#### Phase 3：技能注册表（1-2天）

**文件**：`internal/skills/registry.go`

```go
type SkillRegistry struct {
    mu      sync.RWMutex
    skills  map[string]*Skill
    index   *SkillIndex
    storage storage.Storage
}

func NewSkillRegistry(storage storage.Storage) *SkillRegistry {
    return &SkillRegistry{
        skills:  make(map[string]*Skill),
        storage: storage,
    }
}

func (r *SkillRegistry) Register(skill *Skill) error {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, exists := r.skills[skill.ID]; exists {
        return fmt.Errorf("skill %s already registered", skill.ID)
    }

    r.skills[skill.ID] = skill
    r.index.Add(skill)

    return nil
}

func (r *SkillRegistry) Get(id string) (*Skill, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    skill, exists := r.skills[id]
    return skill, exists
}

func (r *SkillRegistry) List() []*Skill {
    r.mu.RLock()
    defer r.mu.RUnlock()

    skills := make([]*Skill, 0, len(r.skills))
    for _, skill := range r.skills {
        if skill.Enabled {
            skills = append(skills, skill)
        }
    }
    return skills
}

func (r *SkillRegistry) Search(query string) []*Skill {
    r.mu.RLock()
    defer r.mu.RUnlock()

    return r.index.Search(query)
}
```

#### Phase 4：技能选择器（2-3天）

**文件**：`internal/skills/selector.go`

```go
type SkillSelector struct {
    registry *SkillRegistry
    llm      LLMProvider
}

func NewSkillSelector(registry *SkillRegistry, llm LLMProvider) *SkillSelector {
    return &SkillSelector{
        registry: registry,
        llm:      llm,
    }
}

func (s *SkillSelector) Select(ctx context.Context, userMessage string) ([]*Skill, error) {
    keywords := extractKeywords(userMessage)

    candidates := make([]*Skill, 0)

    for _, skill := range s.registry.List() {
        score := s.calculateScore(skill, keywords, userMessage)
        if score > 0.5 {
            candidates = append(candidates, skill)
        }
    }

    if len(candidates) > 3 {
        candidates = s.rankByLLM(ctx, candidates, userMessage)
    }

    return candidates, nil
}

func (s *SkillSelector) calculateScore(skill *Skill, keywords []string, message string) float64 {
    var score float64

    for _, keyword := range keywords {
        if contains(skill.Name, keyword) {
            score += 0.3
        }
        if contains(skill.Description, keyword) {
            score += 0.2
        }
        for _, tag := range skill.Tags {
            if contains(tag, keyword) {
                score += 0.1
            }
        }
    }

    return math.Min(score, 1.0)
}
```

#### Phase 5：文件监控和热更新（2-3天）

**文件**：`internal/skills/watcher.go`

```go
type SkillFileWatcher struct {
    registry *SkillRegistry
    parser   *SkillParser
    watcher  *fsnotify.Watcher
    ctx      context.Context
    cancel   context.CancelFunc
}

func NewSkillFileWatcher(registry *SkillRegistry, parser *SkillParser) (*SkillFileWatcher, error) {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return nil, err
    }

    ctx, cancel := context.WithCancel(context.Background())

    return &SkillFileWatcher{
        registry: registry,
        parser:   parser,
        watcher:  watcher,
        ctx:      ctx,
        cancel:   cancel,
    }, nil
}

func (w *SkillFileWatcher) Watch(path string) error {
    if err := w.watcher.Add(path); err != nil {
        return err
    }

    go w.processEvents()
    return nil
}

func (w *SkillFileWatcher) processEvents() {
    for {
        select {
        case <-w.ctx.Done():
            return
        case event, ok := <-w.watcher.Events:
            if !ok {
                return
            }

            if event.Op&fsnotify.Write == fsnotify.Write ||
               event.Op&fsnotify.Create == fsnotify.Create {
                w.handleFileChange(event.Name)
            }
        case err, ok := <-w.watcher.Errors:
            if !ok {
                return
            }
            log.Printf("File watcher error: %v", err)
        }
    }
}

func (w *SkillFileWatcher) handleFileChange(path string) {
    skill, err := w.parser.Parse(w.ctx, path)
    if err != nil {
        log.Printf("Failed to parse skill file %s: %v", path, err)
        return
    }

    if err := w.registry.Register(skill); err != nil {
        log.Printf("Failed to register skill %s: %v", skill.ID, err)
        return
    }

    log.Printf("Skill %s updated from file: %s", skill.ID, path)
}
```

#### Phase 6：集成到 Agent（1-2天）

**修改**：`internal/agent/agent.go`

```go
type Agent struct {
    messageBus     bus.MessageBus
    llmManager     *llm.MultiModelManager
    toolExecutor   *tools.ToolExecutor
    contextBuilder *agentcontext.Builder
    skillSelector  *skills.SkillSelector
    // ... 其他字段
}

func (a *Agent) HandleMessage(ctx context.Context, msg *bus.Message) error {
    selectedSkills, err := a.skillSelector.Select(ctx, msg.Content)
    if err != nil {
        log.Printf("Failed to select skills: %v", err)
    }

    if len(selectedSkills) > 0 {
        skillContext := a.buildSkillContext(selectedSkills)
        systemPrompt += skillContext
    }

    // ... 继续原有的 ReAct 循环
}
```

### 2.5 配置扩展

**文件**：`configs/config.yaml`

```yaml
skills:
  enabled: true
  directory: "./data/skills"
  auto_reload: true
  max_active: 5
  selection:
    method: "hybrid"  # keyword, llm, hybrid
    threshold: 0.5
```

---

## 3. MCP 接入实现计划

### 3.1 功能概述

MCP (Model Context Protocol) 是一个开放标准，允许 LLM 应用程序与外部工具和数据源进行标准化交互。

### 3.2 核心组件

| 组件 | 文件路径 | 职责 |
|------|----------|------|
| `MCPClient` | `internal/mcp/client.go` | MCP 客户端 |
| `MCPServer` | `internal/mcp/server.go` | MCP 服务器（可选） |
| `MCPToolAdapter` | `internal/mcp/adapter.go` | MCP 工具适配器 |
| `MCPRegistry` | `internal/mcp/registry.go` | MCP 服务器注册表 |
| `MCPProtocol` | `internal/mcp/protocol.go` | MCP 协议实现 |

### 3.3 MCP 协议核心

```go
type MCPMessage struct {
    JSONRPC string      `json:"jsonrpc"`
    ID      interface{} `json:"id,omitempty"`
    Method  string      `json:"method,omitempty"`
    Params  interface{} `json:"params,omitempty"`
    Result  interface{} `json:"result,omitempty"`
    Error   *MCPError   `json:"error,omitempty"`
}

type MCPError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}

type MCPTool struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    InputSchema map[string]interface{} `json:"inputSchema"`
}

type MCPResource struct {
    URI         string `json:"uri"`
    Name        string `json:"name"`
    Description string `json:"description"`
    MimeType    string `json:"mimeType"`
}
```

### 3.4 实现步骤

#### Phase 1：MCP 协议基础（2-3天）

**文件**：`internal/mcp/protocol.go`

```go
type MCPProtocol struct {
    client *http.Client
}

func NewMCPProtocol() *MCPProtocol {
    return &MCPProtocol{
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (p *MCPProtocol) SendRequest(ctx context.Context, endpoint string, req *MCPMessage) (*MCPMessage, error) {
    body, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }

    httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
    if err != nil {
        return nil, err
    }

    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := p.client.Do(httpReq)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("MCP server returned status %d", resp.StatusCode)
    }

    var mcpResp MCPMessage
    if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
        return nil, err
    }

    return &mcpResp, nil
}

func (p *MCPProtocol) ListTools(ctx context.Context, endpoint string) ([]MCPTool, error) {
    req := &MCPMessage{
        JSONRPC: "2.0",
        ID:      generateID(),
        Method:  "tools/list",
    }

    resp, err := p.SendRequest(ctx, endpoint, req)
    if err != nil {
        return nil, err
    }

    if resp.Error != nil {
        return nil, fmt.Errorf("MCP error: %s", resp.Error.Message)
    }

    result, ok := resp.Result.(map[string]interface{})
    if !ok {
        return nil, fmt.Errorf("invalid response format")
    }

    toolsData, ok := result["tools"].([]interface{})
    if !ok {
        return nil, fmt.Errorf("invalid tools format")
    }

    tools := make([]MCPTool, 0, len(toolsData))
    for _, toolData := range toolsData {
        toolMap, ok := toolData.(map[string]interface{})
        if !ok {
            continue
        }

        tool := MCPTool{
            Name:        getString(toolMap, "name"),
            Description: getString(toolMap, "description"),
            InputSchema: getMap(toolMap, "inputSchema"),
        }
        tools = append(tools, tool)
    }

    return tools, nil
}

func (p *MCPProtocol) CallTool(ctx context.Context, endpoint, toolName string, args map[string]interface{}) (interface{}, error) {
    req := &MCPMessage{
        JSONRPC: "2.0",
        ID:      generateID(),
        Method:  "tools/call",
        Params: map[string]interface{}{
            "name":      toolName,
            "arguments": args,
        },
    }

    resp, err := p.SendRequest(ctx, endpoint, req)
    if err != nil {
        return nil, err
    }

    if resp.Error != nil {
        return nil, fmt.Errorf("MCP error: %s", resp.Error.Message)
    }

    result, ok := resp.Result.(map[string]interface{})
    if !ok {
        return nil, fmt.Errorf("invalid response format")
    }

    content, ok := result["content"].([]interface{})
    if !ok {
        return nil, fmt.Errorf("invalid content format")
    }

    return content, nil
}
```

#### Phase 2：MCP 客户端（2-3天）

**文件**：`internal/mcp/client.go`

```go
type MCPClient struct {
    protocol *MCPProtocol
    endpoint string
    name     string
    tools    map[string]MCPTool
    mu       sync.RWMutex
}

func NewMCPClient(name, endpoint string) *MCPClient {
    return &MCPClient{
        protocol: NewMCPProtocol(),
        endpoint: endpoint,
        name:     name,
        tools:    make(map[string]MCPTool),
    }
}

func (c *MCPClient) Connect(ctx context.Context) error {
    tools, err := c.protocol.ListTools(ctx, c.endpoint)
    if err != nil {
        return fmt.Errorf("failed to list tools: %w", err)
    }

    c.mu.Lock()
    defer c.mu.Unlock()

    for _, tool := range tools {
        c.tools[tool.Name] = tool
    }

    log.Printf("MCP client %s connected with %d tools", c.name, len(tools))
    return nil
}

func (c *MCPClient) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
    c.mu.RLock()
    _, exists := c.tools[toolName]
    c.mu.RUnlock()

    if !exists {
        return "", fmt.Errorf("tool %s not found", toolName)
    }

    result, err := c.protocol.CallTool(ctx, c.endpoint, toolName, args)
    if err != nil {
        return "", err
    }

    return formatResult(result), nil
}

func (c *MCPClient) GetTools() []MCPTool {
    c.mu.RLock()
    defer c.mu.RUnlock()

    tools := make([]MCPTool, 0, len(c.tools))
    for _, tool := range c.tools {
        tools = append(tools, tool)
    }
    return tools
}
```

#### Phase 3：MCP 工具适配器（2-3天）

**文件**：`internal/mcp/adapter.go`

```go
type MCPToolAdapter struct {
    client *MCPClient
    tool   MCPTool
}

func NewMCPToolAdapter(client *MCPClient, tool MCPTool) *MCPToolAdapter {
    return &MCPToolAdapter{
        client: client,
        tool:   tool,
    }
}

func (a *MCPToolAdapter) Name() string {
    return a.tool.Name
}

func (a *MCPToolAdapter) Description() string {
    return a.tool.Description
}

func (a *MCPToolAdapter) Parameters() json.RawMessage {
    schema, err := json.Marshal(a.tool.InputSchema)
    if err != nil {
        return json.RawMessage(`{"type": "object"}`)
    }
    return schema
}

func (a *MCPToolAdapter) Execute(ctx context.Context, params map[string]interface{}) (string, error) {
    return a.client.CallTool(ctx, a.tool.Name, params)
}
```

#### Phase 4：MCP 注册表（1-2天）

**文件**：`internal/mcp/registry.go`

```go
type MCPRegistry struct {
    mu      sync.RWMutex
    clients map[string]*MCPClient
}

func NewMCPRegistry() *MCPRegistry {
    return &MCPRegistry{
        clients: make(map[string]*MCPClient),
    }
}

func (r *MCPRegistry) Register(name, endpoint string) (*MCPClient, error) {
    r.mu.Lock()
    defer r.mu.Unlock()

    if _, exists := r.clients[name]; exists {
        return nil, fmt.Errorf("MCP client %s already registered", name)
    }

    client := NewMCPClient(name, endpoint)
    r.clients[name] = client

    return client, nil
}

func (r *MCPRegistry) Get(name string) (*MCPClient, bool) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    client, exists := r.clients[name]
    return client, exists
}

func (r *MCPRegistry) List() []*MCPClient {
    r.mu.RLock()
    defer r.mu.RUnlock()

    clients := make([]*MCPClient, 0, len(r.clients))
    for _, client := range r.clients {
        clients = append(clients, client)
    }
    return clients
}

func (r *MCPRegistry) ConvertToTools(toolRegistry *tools.ToolRegistry) error {
    for _, client := range r.List() {
        for _, tool := range client.GetTools() {
            adapter := NewMCPToolAdapter(client, tool)
            if err := toolRegistry.Register(adapter); err != nil {
                log.Printf("Failed to register MCP tool %s: %v", tool.Name, err)
            }
        }
    }
    return nil
}
```

#### Phase 5：集成到主程序（1-2天）

**修改**：`cmd/main.go`

```go
func initializeMCP(ctx context.Context, cfg *config.Config, toolRegistry *tools.ToolRegistry) (*mcp.MCPRegistry, error) {
    mcpRegistry := mcp.NewMCPRegistry()

    for _, mcpConfig := range cfg.MCP.Servers {
        client, err := mcpRegistry.Register(mcpConfig.Name, mcpConfig.Endpoint)
        if err != nil {
            log.Printf("Failed to register MCP client %s: %v", mcpConfig.Name, err)
            continue
        }

        if err := client.Connect(ctx); err != nil {
            log.Printf("Failed to connect to MCP server %s: %v", mcpConfig.Name, err)
            continue
        }
    }

    if err := mcpRegistry.ConvertToTools(toolRegistry); err != nil {
        log.Printf("Failed to convert MCP tools: %v", err)
    }

    return mcpRegistry, nil
}
```

### 3.5 配置扩展

**文件**：`configs/config.yaml`

```yaml
mcp:
  enabled: true
  servers:
    - name: "filesystem"
      endpoint: "http://localhost:3000/mcp"
      enabled: true
    - name: "database"
      endpoint: "http://localhost:3001/mcp"
      enabled: false
  timeout: 30
  retry_attempts: 3
```

---

## 4. 定时任务实现计划

### 4.1 功能概述

定时任务系统允许用户通过 Cron 表达式定义定期执行的任务，支持任务创建、删除、切换和手动执行。

### 4.2 核心组件

| 组件 | 文件路径 | 职责 |
|------|----------|------|
| `ScheduledTask` | `internal/schedule/task.go` | 定时任务数据结构 |
| `ScheduledTaskService` | `internal/schedule/service.go` | 任务调度服务 |
| `ScheduledTaskExecutor` | `internal/schedule/executor.go` | 任务执行器 |
| `CronParser` | `internal/schedule/cron.go` | Cron 表达式解析器 |
| `TaskStorage` | `internal/schedule/storage.go` | 任务持久化 |

### 4.3 数据结构

```go
type ScheduledTask struct {
    ID           string            `json:"id"`
    Name         string            `json:"name"`
    Description  string            `json:"description"`
    CronExpr     string            `json:"cron_expr"`
    Prompt       string            `json:"prompt"`
    Enabled      bool              `json:"enabled"`
    LastRunAt    *time.Time        `json:"last_run_at"`
    NextRunAt    *time.Time        `json:"next_run_at"`
    LastSuccess  bool              `json:"last_success"`
    LastError    string            `json:"last_error,omitempty"`
    CreatedAt    time.Time         `json:"created_at"`
    UpdatedAt    time.Time         `json:"updated_at"`
    Metadata     map[string]string `json:"metadata"`
}

type TaskExecution struct {
    ID        string    `json:"id"`
    TaskID    string    `json:"task_id"`
    StartedAt time.Time `json:"started_at"`
    EndedAt   *time.Time `json:"ended_at,omitempty"`
    Success   bool      `json:"success"`
    Output    string    `json:"output"`
    Error     string    `json:"error,omitempty"`
}
```

### 4.4 实现步骤

#### Phase 1：Cron 表达式解析器（2-3天）

**文件**：`internal/schedule/cron.go`

```go
type CronSchedule struct {
    minute  []int
    hour    []int
    dom     []int
    month   []int
    dow     []int
}

type CronParser struct{}

func NewCronParser() *CronParser {
    return &CronParser{}
}

func (p *CronParser) Parse(expr string) (*CronSchedule, error) {
    parts := strings.Split(expr, " ")
    if len(parts) != 5 {
        return nil, fmt.Errorf("invalid cron expression: expected 5 parts, got %d", len(parts))
    }

    schedule := &CronSchedule{}

    var err error
    schedule.minute, err = p.parseField(parts[0], 0, 59)
    if err != nil {
        return nil, fmt.Errorf("invalid minute field: %w", err)
    }

    schedule.hour, err = p.parseField(parts[1], 0, 23)
    if err != nil {
        return nil, fmt.Errorf("invalid hour field: %w", err)
    }

    schedule.dom, err = p.parseField(parts[2], 1, 31)
    if err != nil {
        return nil, fmt.Errorf("invalid day of month field: %w", err)
    }

    schedule.month, err = p.parseField(parts[3], 1, 12)
    if err != nil {
        return nil, fmt.Errorf("invalid month field: %w", err)
    }

    schedule.dow, err = p.parseField(parts[4], 0, 6)
    if err != nil {
        return nil, fmt.Errorf("invalid day of week field: %w", err)
    }

    return schedule, nil
}

func (p *CronParser) parseField(field string, min, max int) ([]int, error) {
    values := make([]int, 0)

    parts := strings.Split(field, ",")
    for _, part := range parts {
        nums, err := p.parsePart(part, min, max)
        if err != nil {
            return nil, err
        }
        values = append(values, nums...)
    }

    sort.Ints(values)
    return values, nil
}

func (p *CronParser) parsePart(part string, min, max int) ([]int, error) {
    if strings.Contains(part, "/") {
        return p.parseStep(part, min, max)
    } else if strings.Contains(part, "-") {
        return p.parseRange(part, min, max)
    } else if part == "*" {
        return p.generateRange(min, max), nil
    } else {
        num, err := strconv.Atoi(part)
        if err != nil {
            return nil, fmt.Errorf("invalid number: %s", part)
        }
        if num < min || num > max {
            return nil, fmt.Errorf("number %d out of range [%d, %d]", num, min, max)
        }
        return []int{num}, nil
    }
}

func (s *CronSchedule) Next(from time.Time) time.Time {
    t := from.Add(time.Minute).Truncate(time.Minute)

    for {
        if s.matches(t) {
            return t
        }
        t = t.Add(time.Minute)
    }
}

func (s *CronSchedule) matches(t time.Time) bool {
    if !containsInt(s.minute, t.Minute()) {
        return false
    }
    if !containsInt(s.hour, t.Hour()) {
        return false
    }
    if !containsInt(s.dom, t.Day()) {
        return false
    }
    if !containsInt(s.month, int(t.Month())) {
        return false
    }
    if !containsInt(s.dow, int(t.Weekday())) {
        return false
    }
    return true
}
```

#### Phase 2：任务存储（1-2天）

**文件**：`internal/schedule/storage.go`

```go
type TaskStorage struct {
    storage storage.Storage
    mu      sync.RWMutex
}

func NewTaskStorage(storage storage.Storage) *TaskStorage {
    return &TaskStorage{
        storage: storage,
    }
}

func (s *TaskStorage) Save(ctx context.Context, task *ScheduledTask) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    data, err := json.Marshal(task)
    if err != nil {
        return err
    }

    path := fmt.Sprintf("schedule/tasks/%s.json", task.ID)
    return s.storage.WriteFile(ctx, path, data)
}

func (s *TaskStorage) Get(ctx context.Context, id string) (*ScheduledTask, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    path := fmt.Sprintf("schedule/tasks/%s.json", id)
    data, err := s.storage.ReadFile(ctx, path)
    if err != nil {
        return nil, err
    }

    var task ScheduledTask
    if err := json.Unmarshal(data, &task); err != nil {
        return nil, err
    }

    return &task, nil
}

func (s *TaskStorage) List(ctx context.Context) ([]*ScheduledTask, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    files, err := s.storage.ListFiles(ctx, "schedule/tasks")
    if err != nil {
        return nil, err
    }

    tasks := make([]*ScheduledTask, 0, len(files))
    for _, file := range files {
        data, err := s.storage.ReadFile(ctx, file)
        if err != nil {
            continue
        }

        var task ScheduledTask
        if err := json.Unmarshal(data, &task); err != nil {
            continue
        }

        tasks = append(tasks, &task)
    }

    return tasks, nil
}

func (s *TaskStorage) Delete(ctx context.Context, id string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    path := fmt.Sprintf("schedule/tasks/%s.json", id)
    return s.storage.DeleteFile(ctx, path)
}
```

#### Phase 3：任务执行器（2-3天）

**文件**：`internal/schedule/executor.go`

```go
type TaskExecutor struct {
    agent       *agent.Agent
    storage     *TaskStorage
    messageBus  bus.MessageBus
    execStorage *ExecutionStorage
}

func NewTaskExecutor(agent *agent.Agent, storage *TaskStorage, messageBus bus.MessageBus) *TaskExecutor {
    return &TaskExecutor{
        agent:       agent,
        storage:     storage,
        messageBus:  messageBus,
        execStorage: NewExecutionStorage(storage.storage),
    }
}

func (e *TaskExecutor) Execute(ctx context.Context, task *ScheduledTask) (*TaskExecution, error) {
    execution := &TaskExecution{
        ID:        generateExecutionID(),
        TaskID:    task.ID,
        StartedAt: time.Now(),
    }

    msg := &bus.Message{
        ID:      fmt.Sprintf("scheduled-%s", execution.ID),
        Channel: bus.ChannelCLI,
        ChatID:  fmt.Sprintf("scheduled-%s", task.ID),
        Content: task.Prompt,
    }

    if err := e.messageBus.Publish(ctx, bus.ChannelCLI, msg); err != nil {
        execution.Success = false
        execution.Error = err.Error()
        return execution, err
    }

    now := time.Now()
    execution.EndedAt = &now
    execution.Success = true

    task.LastRunAt = &now
    task.LastSuccess = true
    task.LastError = ""

    if err := e.storage.Save(ctx, task); err != nil {
        log.Printf("Failed to update task %s: %v", task.ID, err)
    }

    if err := e.execStorage.Save(ctx, execution); err != nil {
        log.Printf("Failed to save execution %s: %v", execution.ID, err)
    }

    return execution, nil
}
```

#### Phase 4：任务调度服务（3-4天）

**文件**：`internal/schedule/service.go`

```go
type ScheduledTaskService struct {
    storage   *TaskStorage
    executor  *TaskExecutor
    parser    *CronParser
    tasks     map[string]*ScheduledTask
    timers    map[string]*time.Timer
    mu        sync.RWMutex
    ctx       context.Context
    cancel    context.CancelFunc
    messageBus bus.MessageBus
}

func NewScheduledTaskService(storage *TaskStorage, executor *TaskExecutor, messageBus bus.MessageBus) *ScheduledTaskService {
    ctx, cancel := context.WithCancel(context.Background())

    return &ScheduledTaskService{
        storage:    storage,
        executor:   executor,
        parser:     NewCronParser(),
        tasks:      make(map[string]*ScheduledTask),
        timers:     make(map[string]*time.Timer),
        ctx:        ctx,
        cancel:     cancel,
        messageBus: messageBus,
    }
}

func (s *ScheduledTaskService) Start() error {
    tasks, err := s.storage.List(s.ctx)
    if err != nil {
        return fmt.Errorf("failed to load tasks: %w", err)
    }

    for _, task := range tasks {
        if err := s.scheduleTask(task); err != nil {
            log.Printf("Failed to schedule task %s: %v", task.ID, err)
        }
    }

    go s.run()

    return nil
}

func (s *ScheduledTaskService) Stop() {
    s.cancel()

    s.mu.Lock()
    defer s.mu.Unlock()

    for _, timer := range s.timers {
        timer.Stop()
    }
}

func (s *ScheduledTaskService) Create(ctx context.Context, name, description, cronExpr, prompt string) (*ScheduledTask, error) {
    schedule, err := s.parser.Parse(cronExpr)
    if err != nil {
        return nil, fmt.Errorf("invalid cron expression: %w", err)
    }

    task := &ScheduledTask{
        ID:          generateTaskID(),
        Name:        name,
        Description: description,
        CronExpr:    cronExpr,
        Prompt:      prompt,
        Enabled:     true,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }

    task.NextRunAt = ptrTo(schedule.Next(time.Now()))

    if err := s.storage.Save(ctx, task); err != nil {
        return nil, err
    }

    if err := s.scheduleTask(task); err != nil {
        return nil, err
    }

    return task, nil
}

func (s *ScheduledTaskService) Delete(ctx context.Context, id string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if timer, exists := s.timers[id]; exists {
        timer.Stop()
        delete(s.timers, id)
    }

    delete(s.tasks, id)

    return s.storage.Delete(ctx, id)
}

func (s *ScheduledTaskService) Toggle(ctx context.Context, id string, enabled bool) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    task, exists := s.tasks[id]
    if !exists {
        return fmt.Errorf("task %s not found", id)
    }

    task.Enabled = enabled
    task.UpdatedAt = time.Now()

    if enabled {
        if err := s.scheduleTask(task); err != nil {
            return err
        }
    } else {
        if timer, exists := s.timers[id]; exists {
            timer.Stop()
            delete(s.timers, id)
        }
    }

    return s.storage.Save(ctx, task)
}

func (s *ScheduledTaskService) RunNow(ctx context.Context, id string) (*TaskExecution, error) {
    s.mu.RLock()
    task, exists := s.tasks[id]
    s.mu.RUnlock()

    if !exists {
        return nil, fmt.Errorf("task %s not found", id)
    }

    return s.executor.Execute(ctx, task)
}

func (s *ScheduledTaskService) scheduleTask(task *ScheduledTask) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if !task.Enabled {
        return nil
    }

    if timer, exists := s.timers[task.ID]; exists {
        timer.Stop()
    }

    if task.NextRunAt == nil {
        schedule, err := s.parser.Parse(task.CronExpr)
        if err != nil {
            return err
        }
        next := schedule.Next(time.Now())
        task.NextRunAt = &next
    }

    duration := time.Until(*task.NextRunAt)
    if duration <= 0 {
        duration = time.Second
    }

    timer := time.AfterFunc(duration, func() {
        s.executeTask(task)
    })

    s.tasks[task.ID] = task
    s.timers[task.ID] = timer

    return nil
}

func (s *ScheduledTaskService) executeTask(task *ScheduledTask) {
    execution, err := s.executor.Execute(s.ctx, task)
    if err != nil {
        log.Printf("Failed to execute task %s: %v", task.ID, err)
        task.LastSuccess = false
        task.LastError = err.Error()
    } else {
        log.Printf("Task %s executed successfully", task.ID)
        task.LastSuccess = true
        task.LastError = ""
    }

    schedule, err := s.parser.Parse(task.CronExpr)
    if err != nil {
        log.Printf("Failed to parse cron expression for task %s: %v", task.ID, err)
        return
    }

    next := schedule.Next(time.Now())
    task.NextRunAt = &next

    s.scheduleTask(task)
}

func (s *ScheduledTaskService) run() {
    ticker := time.NewTicker(time.Minute)
    defer ticker.Stop()

    for {
        select {
        case <-s.ctx.Done():
            return
        case <-ticker.C:
            s.checkPendingTasks()
        }
    }
}

func (s *ScheduledTaskService) checkPendingTasks() {
    s.mu.RLock()
    defer s.mu.RUnlock()

    now := time.Now()

    for _, task := range s.tasks {
        if task.Enabled && task.NextRunAt != nil && task.NextRunAt.Before(now) {
            go s.executeTask(task)
        }
    }
}
```

#### Phase 5：集成到主程序（1-2天）

**修改**：`cmd/main.go`

```go
var (
    taskService *schedule.ScheduledTaskService
)

func initializeSchedule(ctx context.Context, messageBus bus.MessageBus, agentService *agent.Agent, storage storage.Storage) (*schedule.ScheduledTaskService, error) {
    taskStorage := schedule.NewTaskStorage(storage)
    executor := schedule.NewTaskExecutor(agentService, taskStorage, messageBus)
    taskService := schedule.NewScheduledTaskService(taskStorage, executor, messageBus)

    if err := taskService.Start(); err != nil {
        return nil, fmt.Errorf("failed to start task service: %w", err)
    }

    log.Printf("Scheduled task service started")
    return taskService, nil
}
```

### 4.5 配置扩展

**文件**：`configs/config.yaml`

```yaml
schedule:
  enabled: true
  storage_path: "./data/schedule"
  max_executions: 100
  execution_retention_days: 30
  tasks:
    - name: "daily_summary"
      description: "Generate daily summary"
      cron_expr: "0 9 * * *"
      prompt: "Generate a summary of today's activities"
      enabled: false
```

---

## 5. 实施路线图

### 5.1 总体时间规划

| 阶段 | 功能 | 预计时间 | 优先级 |
|------|------|----------|--------|
| 阶段 1 | Skills 系统 | 10-15 天 | 高 |
| 阶段 2 | MCP 接入 | 8-12 天 | 中 |
| 阶段 3 | 定时任务 | 10-14 天 | 中 |
| 阶段 4 | 集成测试 | 5-7 天 | 高 |
| 阶段 5 | 文档编写 | 3-5 天 | 中 |

**总计**：36-53 天

### 5.2 详细时间表

#### 第 1-2 周：Skills 系统 Phase 1-3

- Week 1:
  - Day 1-2: 技能数据结构定义
  - Day 3-4: 技能解析器实现
  - Day 5: 单元测试编写

- Week 2:
  - Day 1-2: 技能注册表实现
  - Day 3-4: 技能选择器实现
  - Day 5: 集成测试

#### 第 3-4 周：Skills 系统 Phase 4-6 + MCP Phase 1-2

- Week 3:
  - Day 1-2: 文件监控和热更新
  - Day 3-4: 集成到 Agent
  - Day 5: Skills 系统测试

- Week 4:
  - Day 1-3: MCP 协议基础实现
  - Day 4-5: MCP 客户端实现

#### 第 5-6 周：MCP Phase 3-5 + 定时任务 Phase 1-2

- Week 5:
  - Day 1-2: MCP 工具适配器
  - Day 3: MCP 注册表
  - Day 4-5: 集成到主程序

- Week 6:
  - Day 1-3: Cron 表达式解析器
  - Day 4-5: 任务存储实现

#### 第 7-8 周：定时任务 Phase 3-5

- Week 7:
  - Day 1-2: 任务执行器
  - Day 3-4: 任务调度服务
  - Day 5: 单元测试

- Week 8:
  - Day 1-2: 集成到主程序
  - Day 3-5: 定时任务测试

#### 第 9-10 周：集成测试和文档

- Week 9:
  - Day 1-3: 端到端集成测试
  - Day 4-5: 性能测试

- Week 10:
  - Day 1-3: 文档编写
  - Day 4-5: 代码审查和优化

### 5.3 里程碑

| 里程碑 | 日期 | 交付物 |
|--------|------|--------|
| M1 | 第 2 周末 | Skills 系统 MVP 完成 |
| M2 | 第 4 周末 | MCP 基础功能完成 |
| M3 | 第 6 周末 | 定时任务基础功能完成 |
| M4 | 第 8 周末 | 所有功能开发完成 |
| M5 | 第 10 周末 | 测试和文档完成 |

---

## 6. 测试策略

### 6.1 单元测试

每个模块都需要编写单元测试：

- **Skills 系统**：
  - 技能解析器测试
  - 技能注册表测试
  - 技能选择器测试
  - 文件监控测试

- **MCP 接入**：
  - MCP 协议测试
  - MCP 客户端测试
  - 工具适配器测试

- **定时任务**：
  - Cron 解析器测试
  - 任务存储测试
  - 任务执行器测试
  - 任务调度服务测试

### 6.2 集成测试

测试各模块之间的交互：

- Skills 与 Agent 的集成
- MCP 与工具系统的集成
- 定时任务与 Agent 的集成

### 6.3 端到端测试

测试完整的用户流程：

1. 创建技能文件 → Agent 自动选择并使用技能
2. 注册 MCP 服务器 → Agent 调用 MCP 工具
3. 创建定时任务 → 任务按时执行

### 6.4 性能测试

- Skills 选择性能（大量技能场景）
- MCP 工具调用延迟
- 定时任务调度准确性

### 6.5 测试覆盖率目标

- 单元测试覆盖率：≥ 80%
- 集成测试覆盖率：≥ 70%
- 端到端测试：覆盖主要用户场景

---

## 7. 风险和缓解措施

### 7.1 技术风险

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| Cron 解析复杂度 | 高 | 中 | 使用成熟的 Cron 库 |
| MCP 协议变更 | 中 | 低 | 版本化协议实现 |
| 文件监控性能 | 中 | 中 | 优化事件处理逻辑 |
| 技能选择准确性 | 高 | 中 | 结合关键词和 LLM |

### 7.2 进度风险

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| 开发时间超期 | 高 | 中 | 分阶段交付 |
| 测试时间不足 | 中 | 中 | 提前开始测试 |
| 文档编写延迟 | 低 | 高 | 并行编写文档 |

### 7.3 质量风险

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| 代码质量问题 | 高 | 中 | 代码审查 |
| 测试覆盖率不足 | 中 | 中 | 强制测试覆盖率 |
| 性能问题 | 中 | 低 | 性能测试和优化 |

---

## 8. 依赖项

### 8.1 新增 Go 依赖

```go
// Skills 系统
github.com/fsnotify/fsnotify  // 文件监控
gopkg.in/yaml.v3              // YAML 解析

// MCP 接入
// 无额外依赖，使用标准库

// 定时任务
// 无额外依赖，使用标准库
```

### 8.2 更新 go.mod

```go
module github.com/wjffsx/miniclaw_go

go 1.25.1

require (
    github.com/fsnotify/fsnotify v1.7.0
    github.com/gorilla/websocket v1.5.3
    gopkg.in/yaml.v3 v3.0.1
)
```

---

## 9. 后续扩展方向

完成这三个功能后，可以考虑以下扩展：

1. **SubAgent 系统**：实现子代理派生和编排
2. **渠道推送**：实现邮件/Webhook 推送
3. **远程节点管理**：实现 SSH/K8s 远程执行
4. **审计日志**：实现全链路审计记录
5. **Agent 身份配置**：实现多 Agent 身份和权限控制
6. **文件生成预览**：实现流式文件预览
7. **会话自动命名**：实现 LLM 自动生成会话标题
8. **Web UI**：开发 Web 界面

---

## 10. 总结

本实现计划为 MiniClaw Go 添加了三个核心功能：

1. **Skills 系统**：提供灵活的技能定义和选择机制
2. **MCP 接入**：实现标准化的工具协议支持
3. **定时任务**：提供强大的任务调度能力

通过分阶段实施，预计在 10 周内完成所有功能的开发和测试。每个功能都经过精心设计，与现有架构无缝集成，确保代码质量和可维护性。
