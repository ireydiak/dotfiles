-- Keymaps are automatically loaded on the VeryLazy event
-- Default keymaps that are always set: https://github.com/LazyVim/LazyVim/blob/main/lua/lazyvim/config/keymaps.lua
-- Add any additional keymaps here
--
local map = vim.keymap.set

map("i", "jj", "<ESC>", { noremap = false })

-- which-key keymaps
--vim.api.nvim_set_keymap("n", "<leader>?", "<cmd>WhichKey<cr>", {noremap=true, silent=true, desc="Open which-key"})
map("n", "<leader>?", "<cmd>WhichKey<cr>", { noremap = true, silent = true, desc = "Open which-key" })

-- movement in edit mode
map("i", "<C-b>", "<ESC>^i", { desc = "move beginning of line" })
map("i", "<C-e>", "<End>", { desc = "move end of line" })
map("i", "<C-h>", "<Left>", { desc = "move left" })
map("i", "<C-l>", "<Right>", { desc = "move right" })
map("i", "<C-j>", "<Down>", { desc = "move down" })
map("i", "<C-k>", "<Up>", { desc = "move up" })

-- switch windows in normal mode
map("n", "<C-h>", "<C-w>h", { desc = "switch window left" })
map("n", "<C-l>", "<C-w>l", { desc = "switch window right" })
map("n", "<C-j>", "<C-w>j", { desc = "switch window down" })
map("n", "<C-k>", "<C-w>k", { desc = "switch window up" })

-- Close buffer without closing the window
map("n", "<leader>q", function()
  vim.cmd("write")
  vim.cmd("bdelete")
end, { desc = "Save and close buffer" })

-- These are Neovim key mappings that allow you to move selected text blocks up and down in visual mod
map("v", "K", ":m '<-2<CR>gv=gv") -- Move selection up
map("v", "J", ":m '>+1<CR>gv=gv") -- Move selection down

-- Center after movement
map("n", "<C-d>", "<C-d>zz")
map("n", "<C-u>", "<C-u>zz")

-- Skim through quickfix errors
map("n", "]p", ":cnext<CR>", { desc = "Next quickfix" })
map("n", "[p", ":cprev<CR>", { desc = "Previous quickfix" })
map("n", "<leader>co", ":copen<CR>", { desc = "Open quickfix" })

-- clipboard magic
-- Paste over selection without losing clipboard content
map("x", "<leader>p", [["_dP]])
map({ "n", "v" }, "<leader>y", [["+y]]) -- Copy to system clipboard
map({ "n", "v" }, "<leader>d", [["_d]])

-- search and replace
vim.keymap.set("n", "<leader>sr", function()
  local word = vim.fn.expand("<cword>")
  vim.api.nvim_feedkeys(":%s/" .. word .. "//gI", "n", false)
end, { desc = "Substitute word under cursor" })
