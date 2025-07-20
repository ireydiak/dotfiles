-- Fuzzy finder for files, buffers, grep, and more
return {
	{
		"nvim-telescope/telescope.nvim",
		tag = "0.1.8",
		dependencies = { "nvim-lua/plenary.nvim" },
	},

	{
		"nvim-telescope/telescope-fzf-native.nvim",
		run = "make",
	},

	-- Neovim plugin for splitting/joining blocks of code like arrays, hashes, statements, objects, dictionaries, etc.
	{
		"Wansmer/treesj",
		keys = { "<space>m", "<space>j", "<space>s" },
		dependencies = { "nvim-treesitter/nvim-treesitter" },
	},
}
