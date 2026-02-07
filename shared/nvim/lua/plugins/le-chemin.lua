return {
    "ireydiak/le-chemin.nvim",
    event = "VeryLazy",
    opts = {
        root_markers = { ".git", "package.json", "pyproject.toml", "Makefile" },
    },
    keys = {
        { "<leader>cp", "<cmd>LeCheminRelative<cr>", desc = "Copy Relative Path" },
        { "<leader>cP", "<cmd>LeCheminAbsolute<cr>", desc = "Copy Absolute Path" },
    },
}

