-- Task management skill
-- Usage: /task create "send daily report" "0 9 * * *"

function create_task(args)
    local prompt = args[1]
    local schedule = args[2] or "once"
    
    local id = uuid()
    local sql = string.format(
        "INSERT INTO tasks (id, group_folder, chat_jid, prompt, schedule_type, schedule_value, status, created_at) VALUES ('%s', '%s', '%s', '%s', 'cron', '%s', 'active', datetime('now'))",
        id, GROUP_FOLDER, CHAT_JID, prompt, schedule
    )
    
    local err = db:exec(sql)
    if err then
        log("Error creating task: " .. err)
        return "Failed to create task"
    end
    
    return "Task created: " .. id
end

function list_tasks()
    log("Listing tasks for " .. GROUP_FOLDER)
    return "Tasks listed (implement query)"
end

-- Main entry
if #arg > 0 then
    if arg[1] == "create" then
        return create_task({table.unpack(arg, 2)})
    elseif arg[1] == "list" then
        return list_tasks()
    end
end
