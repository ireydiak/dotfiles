-- UI components: file manager and color themes
return {
	-- file manager
	{
		"nvim-tree/nvim-tree.lua",
		cmd = { "NvimTreeToggle", "NvimTreeFocus" },
		opts = function()
			return require("config.nvimtree")
		end,
	},

	-- themes and colors
	{
		"rose-pine/neovim",
		name = "rose-pine",
		config = function()
			-- set colorscheme
			vim.cmd("colorscheme rose-pine")
		end,
	},

	-- icons
	{
		"nvim-tree/nvim-web-devicons",
		opts = {},
	},
}
