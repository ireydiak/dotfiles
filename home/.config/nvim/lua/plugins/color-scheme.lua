return {
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
    },
    {
        "metalelf0/black-metal-theme-neovim",
        lazy = false,
        priority = 1000,
        config = function()
            require("black-metal").setup({
                theme = "immortal"
            })
            require("black-metal").load()
        end,
    }
}

