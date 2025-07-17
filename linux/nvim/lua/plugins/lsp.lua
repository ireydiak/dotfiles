return {
	{
		"mason-org/mason.nvim",
		opts = {},
	},
	-- formatter
	{
		"stevearc/conform.nvim",
		opts = {
			formatters_by_ft = {
				go = { "gofmt" },
				rust = { "rustfmt", lsp_format = "fallback" },
				javascript = { "deno_fmt", "prettier" },
				typescript = { "deno_fmt", "prettier" },
				lua = { "stylua" },
			},
			format_on_save = { timeout_ms = 500 },
		},
	},
}
