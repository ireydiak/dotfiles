return {
  --{ "shaunsingh/nord.nvim" },
  {
    "rose-pine/neovim",
    name = "rose-pine",
    -- config = function()
    --   vim.cmd("colorscheme rose-pine")
    -- end,
  },
  {
    "catppuccin/nvim",
     name = "catppuccin",
     priority = 1000,
     config = function()
       vim.cmd("colorscheme catppuccin-macchiato")
     end,
  },
  {
    "LazyVim/LazyVim",
    opts = {
      colorscheme = "catppuccin-macchiato",
    },
  },
}

