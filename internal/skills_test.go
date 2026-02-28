package internal

import (
	"os"
	"path/filepath"
	"testing"

)

func TestSkillRegistryXSkip_RegisterAndGet(t *testing.T) {
	db := TestTempDB(t)
	registry := NewSkillRegistry(db)
	defer registry.Close()
	
	skill := &Skill{
		Name:        "test-skill",
		Description: "A test skill",
		Version:     "1.0.0",
	}
	
	registry.Register(skill)
	
	got, ok := registry.Get("test-skill")
	if !ok {
		t.Fatal("Expected to find skill")
	}
	
	if got.Name != skill.Name {
		t.Errorf("Name = %q, want %q", got.Name, skill.Name)
	}
	if got.Description != skill.Description {
		t.Errorf("Description = %q, want %q", got.Description, skill.Description)
	}
}

func TestSkillRegistryXSkip_LoadFromDir(t *testing.T) {
	db := TestTempDB(t)
	registry := NewSkillRegistry(db)
	defer registry.Close()
	
	// 创建临时技能目录
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("Failed to create skill dir: %v", err)
	}
	
	// 创建SKILL.md
	skillContent := `# Test Skill

This is a test skill.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("Failed to write SKILL.md: %v", err)
	}
	
	// 加载技能
	if err := registry.LoadFromDir(tmpDir); err != nil {
		t.Fatalf("LoadFromDir failed: %v", err)
	}
	
	// 验证技能已加载
	got, ok := registry.Get("test-skill")
	if !ok {
		t.Fatal("Expected to find loaded skill")
	}
	
	if got.Name != "test-skill" {
		t.Errorf("Name = %q, want %q", got.Name, "test-skill")
	}
}

func TestSkillRegistryXSkip_LoadFromDir_WithLua(t *testing.T) {
	db := TestTempDB(t)
	registry := NewSkillRegistry(db)
	defer registry.Close()
	
	// 创建临时技能目录
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "lua-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("Failed to create skill dir: %v", err)
	}
	
	// 创建SKILL.md
	skillContent := `# Lua Skill

Skill with Lua script.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("Failed to write SKILL.md: %v", err)
	}
	
	// 创建script.lua
	luaScript := `
function main()
    log("Hello from Lua!")
    return "Success"
end
`
	if err := os.WriteFile(filepath.Join(skillDir, "script.lua"), []byte(luaScript), 0644); err != nil {
		t.Fatalf("Failed to write script.lua: %v", err)
	}
	
	// 加载技能
	if err := registry.LoadFromDir(tmpDir); err != nil {
		t.Fatalf("LoadFromDir failed: %v", err)
	}
	
	// 验证技能已加载
	got, ok := registry.Get("lua-skill")
	if !ok {
		t.Fatal("Expected to find loaded skill")
	}
	
	if got.LuaScript == "" {
		t.Error("Expected LuaScript to be loaded")
	}
}

func TestSkillRegistryXSkip_Execute(t *testing.T) {
	db := TestTempDB(t)
	registry := NewSkillRegistry(db)
	defer registry.Close()
	
	skill := &Skill{
		Name:        "exec-skill",
		Description: "Execution test",
		Steps: []SkillStep{
			{
				Action: "log",
				Params: map[string]string{
					"message": "Test step executed",
				},
			},
		},
	}
	
	registry.Register(skill)
	
	ctx := SkillContext{
		GroupFolder: "test",
		ChatJID:     "test@nanoclaw",
		Args:        map[string]string{},
	}
	
	// 执行技能
	err := registry.Execute(nil, "exec-skill", ctx)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
}

func TestSkillRegistryXSkipFail_LuaBindings(t *testing.T) {
	db := TestTempDB(t)
	registry := NewSkillRegistry(db)
	defer registry.Close()
	
	// 测试log函数
	err := registry.L.DoString(`log("Test log message")`)
	if err != nil {
		t.Errorf("log function failed: %v", err)
	}
	
	// 测试uuid函数
	err = registry.L.DoString(`local id = uuid()`)
	if err != nil {
		t.Errorf("uuid function failed: %v", err)
	}
	
	// 测试db:exec函数（创建表）
	err = registry.L.DoString(`
		local err = db:exec("CREATE TABLE IF NOT EXISTS test_lua (id TEXT PRIMARY KEY)")
		if err then
			log("Error: " .. err)
		end
	`)
	if err != nil {
		t.Errorf("db:exec failed: %v", err)
	}
}

func TestSkillRegistryXSkip_LuaScriptExecution(t *testing.T) {
	db := TestTempDB(t)
	registry := NewSkillRegistry(db)
	defer registry.Close()
	
	skill := &Skill{
		Name: "lua-exec-skill",
		LuaScript: `
			log("Lua script running")
			local id = uuid()
			log("Generated ID: " .. id)
		`,
	}
	
	registry.Register(skill)
	
	ctx := SkillContext{
		GroupFolder: "test",
		ChatJID:     "test@nanoclaw",
	}
	
	err := registry.Execute(nil, "lua-exec-skill", ctx)
	if err != nil {
		t.Fatalf("Execute with Lua failed: %v", err)
	}
}

func TestSkillRegistryXSkip_NotFound(t *testing.T) {
	db := TestTempDB(t)
	registry := NewSkillRegistry(db)
	defer registry.Close()
	
	_, ok := registry.Get("non-existent")
	if ok {
		t.Error("Expected skill not found")
	}
	
	ctx := SkillContext{}
	err := registry.Execute(nil, "non-existent", ctx)
	if err == nil {
		t.Error("Expected error for non-existent skill")
	}
}
