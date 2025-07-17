-- use space characters with 4 spaces for indentation
vim.cmd("set expandtab")
vim.cmd("set tabstop=4")
vim.cmd("set softtabstop=4")
vim.cmd("set shiftwidth=4")

vim.api.nvim_set_keymap("i", "jj", "<ESC>", {noremap=false})
