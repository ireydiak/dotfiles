local M = {}
local active_job = nil

local tools = {
  mypy = {
    cmd = "./bin/mypy",
    efm = "%f:%l: %trror: %m,%f:%l: %tarning: %m,%f:%l: %tote: %m",
  },
}

local function run(tool_name, args)
  local tool = tools[tool_name]
  if not tool then
    vim.notify("pyro-dev: unknown tool " .. tool_name, vim.log.levels.ERROR)
    return
  end

  -- cancel previous run if active
  if active_job then
    vim.fn.jobstop(active_job)
    active_job = nil
  end

  local cmd = { tool.cmd, unpack(args) }
  vim.fn.setqflist({}, "r", { title = tool_name })
  vim.notify(tool_name .. " running...", vim.log.levels.INFO)

  active_job = vim.fn.jobstart(cmd, {
    stdout_buffered = true,
    stderr_buffered = true,
    on_stdout = function(_, data)
      vim.schedule(function()
        vim.fn.setqflist({}, "a", { lines = data, efm = tool.efm })
      end)
    end,
    on_stderr = function(_, data)
      vim.schedule(function()
        vim.fn.setqflist({}, "a", { lines = data, efm = tool.efm })
      end)
    end,
    on_exit = function(_, code)
      active_job = nil
      vim.schedule(function()
        vim.cmd("copen")
        local level = code == 0 and vim.log.levels.INFO or vim.log.levels.WARN
        local count = #vim.fn.getqflist()
        vim.notify(tool_name .. " finished (" .. count .. " entries)", level)
      end)
    end,
  })
end

function M.setup(opts)
  -- merge user tool config if provided
  if opts and opts.tools then
    tools = vim.tbl_deep_extend("force", tools, opts.tools)
  end

  for name, _ in pairs(tools) do
    local cmd_name = name:sub(1, 1):upper() .. name:sub(2)
    vim.api.nvim_create_user_command(cmd_name, function(cmd_opts)
      run(name, cmd_opts.fargs)
    end, { nargs = "*", desc = "Run " .. name })
  end
end

return M
