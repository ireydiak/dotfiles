vim.cmd("set expandtab")
vim.cmd("set tabstop=4")
vim.cmd("set softtabstop=4")
vim.cmd("set shiftwidth=4")
vim.cmd("set autoindent")

require("config.options")
require("config.keymaps")

require("config.lazy")

require("telescope").load_extension("fzf")
