m-- use fzf in telescope
require("telescope").load_extension("fzf")

-- displays which-key gui
vim.api.nvim_set_keymap("n", "<leader>?", require("which-key").show({ global = false }))
