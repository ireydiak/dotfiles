-- Language server protocol support and code formatting
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
				html = { "html_beautfify" },
				typescriptreact = { "prettier" },
			},
			format_on_save = { timeout_ms = 500 },
		},
	},

	--pretty list for showing diagnostics, references, telescope results, quickfix and location lists to help you solve all the trouble your code is causing.
	{
		"folke/trouble.nvim",
		opts = {}, -- for default options, refer to the configuration section for custom setup.
		cmd = "Trouble",
		keys = {
			{
				"<leader>xx",
				"<cmd>Trouble diagnostics toggle<cr>",
				desc = "Diagnostics (Trouble)",
			},
			{
				"<leader>xX",
				"<cmd>Trouble diagnostics toggle filter.buf=0<cr>",
				desc = "Buffer Diagnostics (Trouble)",
			},
			{
				"<leader>cs",
				"<cmd>Trouble symbols toggle focus=false<cr>",
				desc = "Symbols (Trouble)",
			},
			{
				"<leader>cl",
				"<cmd>Trouble lsp toggle focus=false win.position=right<cr>",
				desc = "LSP Definitions / references / ... (Trouble)",
			},
			{
				"<leader>xL",
				"<cmd>Trouble loclist toggle<cr>",
				desc = "Location List (Trouble)",
			},
			{
				"<leader>xQ",
				"<cmd>Trouble qflist toggle<cr>",
				desc = "Quickfix List (Trouble)",
			},
		},
	},
}

