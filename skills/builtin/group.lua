-- Group management skill

function join_group(group_jid)
    log("Joining group: " .. group_jid)
    return "Joined group: " .. group_jid
end

function leave_group(group_jid)
    log("Leaving group: " .. group_jid)
    return "Left group: " .. group_jid
end

if #arg > 0 then
    if arg[1] == "join" then
        return join_group(arg[2])
    elseif arg[1] == "leave" then
        return leave_group(arg[2])
    end
end
