package internal

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/gopher-lua"
)

// Skill 技能定义
type Skill struct {
	Name        string
	Description string
	Version     string
	Steps       []SkillStep
	LuaScript   string
}

// SkillStep 技能步骤
type SkillStep struct {
	Action string
	Params map[string]string
}

// SkillContext 技能执行上下文
type SkillContext struct {
	GroupFolder string
	ChatJID     ChatJID
	Args        map[string]string
}

// SkillRegistry 技能注册表
type SkillRegistry struct {
	skills map[string]*Skill
	L      *lua.LState
	db     *DB
}

// NewSkillRegistry 创建技能注册表
func NewSkillRegistry(db *DB) *SkillRegistry {
	L := lua.NewState()
	sr := &SkillRegistry{
		skills: make(map[string]*Skill),
		L:      L,
		db:     db,
	}
	sr.initLuaBindings()
	return sr
}

// Close 关闭Lua状态
func (sr *SkillRegistry) Close() {
	sr.L.Close()
}

// Register 注册技能
func (sr *SkillRegistry) Register(s *Skill) {
	sr.skills[s.Name] = s
}

// Get 获取技能
func (sr *SkillRegistry) Get(name string) (*Skill, bool) {
	s, ok := sr.skills[name]
	return s, ok
}

// Execute 执行技能
func (sr *SkillRegistry) Execute(ctx context.Context, name string, sc SkillContext) error {
	skill, ok := sr.skills[name]
	if !ok {
		return fmt.Errorf("skill not found: %s", name)
	}

	// 设置上下文变量
	sr.L.SetGlobal("GROUP_FOLDER", lua.LString(sc.GroupFolder))
	sr.L.SetGlobal("CHAT_JID", lua.LString(string(sc.ChatJID)))

	// 执行步骤
	for _, step := range skill.Steps {
		if err := sr.executeStep(step); err != nil {
			return err
		}
	}

	// 执行Lua脚本
	if skill.LuaScript != "" {
		if err := sr.L.DoString(skill.LuaScript); err != nil {
			return fmt.Errorf("lua error: %w", err)
		}
	}

	return nil
}

// LoadFromDir 从目录加载技能
func (sr *SkillRegistry) LoadFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillDir := filepath.Join(dir, entry.Name())
		if err := sr.loadSkill(skillDir); err != nil {
			continue // 跳过无效技能
		}
	}
	return nil
}

// loadSkill 加载单个技能
func (sr *SkillRegistry) loadSkill(dir string) error {
	name := filepath.Base(dir)

	// 读取SKILL.md
	skillPath := filepath.Join(dir, "SKILL.md")
	data, err := os.ReadFile(skillPath)
	if err != nil {
		return err
	}

	skill := &Skill{Name: name}
	// 简化的解析：假设第一行是描述
	lines := strings.Split(string(data), "\n")
	if len(lines) > 0 {
		skill.Description = strings.TrimPrefix(lines[0], "# ")
	}

	// 读取script.lua（可选）
	scriptPath := filepath.Join(dir, "script.lua")
	if scriptData, err := os.ReadFile(scriptPath); err == nil {
		skill.LuaScript = string(scriptData)
	}

	sr.Register(skill)
	return nil
}

// executeStep 执行步骤
func (sr *SkillRegistry) executeStep(step SkillStep) error {
	switch step.Action {
	case "log":
		fmt.Printf("[SKILL] %s\n", step.Params["message"])
	case "db_exec":
		_, err := sr.db.Exec(step.Params["sql"])
		return err
	default:
		return fmt.Errorf("unknown action: %s", step.Action)
	}
	return nil
}

// initLuaBindings 初始化Lua绑定
func (sr *SkillRegistry) initLuaBindings() {
	// 注册db模块
	mod := sr.L.NewTable()
	sr.L.SetField(mod, "exec", sr.L.NewFunction(sr.luaDBExec))
	sr.L.SetField(mod, "query", sr.L.NewFunction(sr.luaDBQuery))
	sr.L.SetGlobal("db", mod)

	// 注册log函数
	sr.L.SetGlobal("log", sr.L.NewFunction(sr.luaLog))

	// 注册uuid函数
	sr.L.SetGlobal("uuid", sr.L.NewFunction(sr.luaUUID))
}

// luaDBExec Lua绑定：执行SQL
func (sr *SkillRegistry) luaDBExec(L *lua.LState) int {
	sql := L.CheckString(1)
	_, err := sr.db.Exec(sql)
	if err != nil {
		L.Push(lua.LString(err.Error()))
		return 1
	}
	L.Push(lua.LNil)
	return 1
}

// luaDBQuery Lua绑定：查询SQL
func (sr *SkillRegistry) luaDBQuery(L *lua.LState) int {
	// 简化实现，返回空表
	L.Push(lua.LString("not implemented"))
	return 1
}

// luaLog Lua绑定：日志
func (sr *SkillRegistry) luaLog(L *lua.LState) int {
	msg := L.CheckString(1)
	fmt.Printf("[LUA] %s\n", msg)
	return 0
}

// luaUUID Lua绑定：生成UUID
func (sr *SkillRegistry) luaUUID(L *lua.LState) int {
	L.Push(lua.LString(fmt.Sprintf("%d", os.Getpid())))
	return 1
}
