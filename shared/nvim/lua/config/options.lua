-- Options are automatically loaded before lazy.nvim startup
-- Default options that are always set: https://github.com/LazyVim/LazyVim/blob/main/lua/lazyvim/config/options.lua
-- Add any additional options here
local opt = vim.opt
local o = vim.o
local g = vim.g

-- tab options
opt.expandtab = true
opt.autoindent = true
opt.tabstop = 4
opt.softtabstop = 4
opt.shiftwidth = 4
opt.smartindent = true

o.cursorline = true
opt.scrolloff = 8
opt.colorcolumn = "90"

-- use system clipboard
o.clipboard = "unnamedplus"

-- set highlight on search
o.hlsearch = true

-- make line numbers default with relative numbering
opt.nu = true
opt.relativenumber = true

-- enable break indents
o.breakindent = true

-- ignore casing in search unless \C or capital in search
o.ignorecase = true
o.smartcase = true

-- set leader key
g.mapleader = " "
g.maplocalleader = " "

-- disable text wrapping
opt.wrap = false

-- disable .swp and backup files
opt.swapfile = false
opt.backup = false

-- search behavior
opt.incsearch = true

-- treats @ and - as valid filename characters
opt.isfname:append("@-@")

-- performance
o.timeoutlen = 400
o.updatetime = 250
opt.termguicolors = true

-- navigation
opt.whichwrap:append("<>[]hl")

-- mason integration
local is_windows = vim.fn.has("win32") ~= 0
local sep = is_windows and "\\" or "/"
local delim = is_windows and ";" or ":"
vim.env.PATH = table.concat({ vim.fn.stdpath("data"), "mason", "bin" }, sep) .. delim .. vim.env.PATH
