-- Autocmds are automatically loaded on the VeryLazy event
-- Default autocmds that are always set: https://github.com/LazyVim/LazyVim/blob/main/lua/lazyvim/config/autocmds.lua
--
-- Add any additional autocmds here
-- with `vim.api.nvim_create_autocmd`
--
-- Or remove existing autocmds by their group name (which is prefixed with `lazyvim_` for the defaults)
-- e.g. vim.api.nvim_del_augroup_by_name("lazyvim_wrap_spell")
vim.api.nvim_create_autocmd("BufWritePre", {
  pattern = { "*.py", "*.js", "*.ts", "*.lua", ".tf" },
  callback = function()
    --if vim.bo.fixeol then
    -- This will add a newline if the file doesn't end with one
    local last_line = vim.fn.line("$")
    if vim.fn.getline(last_line) ~= "" then
      vim.fn.append(last_line, "")
    end
    --end
  end,
})
