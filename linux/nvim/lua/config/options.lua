-- tab options
vim.cmd("set expandtab")
vim.cmd("set tabstop=4")
vim.cmd("set softtabstop=4")
vim.cmd("set shiftwidth=4")
vim.cmd("set autoindent")

-- set highlight on search
vim.o.hlsearch = true

-- make line numbers default with relative numbering
vim.wo.number = true
vim.o.relativenumber = true

-- enable break indents
vim.o.breakindent = true

-- ignore casing in search unless \C or capital in search
vim.o.ignorecase = true
vim.o.smartcase = true

-- use system clipboard
vim.opt.clipboard = "unnamedplus"

-- set leader key
vim.g.mapleader = " "
vim.g.maplocalleader = " "
