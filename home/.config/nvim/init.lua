-- bootstrap lazy.nvim, LazyVim and your plugins
require("config.lazy")

vim.o.fixeol = true

-- add Gleam LSP
vim.lsp.enable('gleam')

