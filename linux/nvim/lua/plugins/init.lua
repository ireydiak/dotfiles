-- Core plugins: search, syntax highlighting, and keybind help
return {

	-- ripgrep
	{
		"doums/rg.nvim",
		cmd = { "Rg", "Rgf", "Rgp", "Rgfp" },
		config = function()
			require("rg").setup({
				excluded = {
					".idea",
					"node_modules",
					".git",
					"target",
					"package-lock.json",
					"Cargo.lock",
				},
			})
		end,
	},

	-- treesitter
	{
		"nvim-treesitter/nvim-treesitter",
		branch = "master",
		lazy = false,
		build = ":TSUpdate",
		opts = function()
			return require("config.treesitter")
		end,
	},

	-- which key
	{
		"folke/which-key.nvim",
		event = "VeryLazy",
		enabled = true,
		event = "VeryLazy",
		opts = {
			preset = "helix",
			debug = vim.uv.cwd():find("which%-key"),
			win = {},
			spec = {},
		},
		keys = {
			{
				"<leader>?",
				function()
					require("which-key").show({ global = false })
				end,
				desc = "Buffer Local Keymaps (which-key)",
			},
		},
	},
}
