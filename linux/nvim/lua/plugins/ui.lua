return {
	{
		"folke/which-key.nvim",
		enabled = true,
		event = "VeryLazy",
		opts = {
			preset = "helix",
			debug = vim.uv.cwd():find("which%-key"),
			win = {},
			spec = {},
		},
	},
}
